package main

import (
	"encoding/json"
	"example-webpage-scrap/model"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
)

const (
	PORT = "8090"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", DefaultHandler).Methods("GET")
	r.HandleFunc("/exam-info", GetExamInfoHandler).Methods("GET")

	addr := fmt.Sprintf(":%s", PORT)
	fmt.Println("Server running on port :", PORT)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	info := model.Response{Message: "running app on " + PORT, Status: true}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}

func GetExamInfoHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "Missing 'url' query parameter", http.StatusBadRequest)
		return
	}

	info, err := ExtractExamInfo(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error extracting exam info: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}

func ExtractExamInfo(url string) (*ExamResult, error) {
	// Fetch the HTML content from the URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	// Check for non-200 HTTP response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	// Parse the HTML with goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// NewFuncToExtarct(doc)
	studentExamInfo := GetStudentExamInfo(doc)
	// SubjectAnalysis := GetExamSujectWiseAnalysis(doc)
	SubjectAnalysis, _ := GetExamSujectWiseAnalysis3(doc)
	// SubjectAnalysisDemo := GetExamSujectWiseAnalysis2(doc)

	return &ExamResult{ExamInfo: studentExamInfo, SubjectAnalysis: SubjectAnalysis}, nil
}

func GetStudentExamInfo(doc *goquery.Document) (model StudentExamInfo) {
	doc.Find(".main-info-pnl table tr").Each(func(index int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() == 2 {
			key := cells.Eq(0).Text()
			value := cells.Eq(1).Text()

			switch key {
			case "Roll Number","Wrong enter-commit direct":
				model.RollNumber = value
			case "Candidate Name":
				model.CandidateName = value
			case "Venue Name", "Test Center Name":
				model.VenueName = value
			case "Exam Date", "Test Date":
				model.ExamDate = value
			case "Exam Time", "Test Time":
				model.ExamTime = value
			case "Subject":
				model.Subject = value
			}
		}
	})
	return model
}

func GetExamSujectWiseAnalysis(doc *goquery.Document) (model []SectionAnalysis) {

	var sectionData []map[string]interface{}
	doc.Find(".grp-cntnr .section-cntnr").Each(func(i int, section *goquery.Selection) {
		label := strings.TrimSpace(section.Find(".section-lbl").Text())

		var tableData []map[string]string
		section.Find("table").Each(func(j int, table *goquery.Selection) {
			examResultInfo := make(map[string]string)

			table.Find("tr").Each(func(k int, row *goquery.Selection) {
				cells := row.Find("td")
				if cells.Length() >= 2 {
					key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(cells.Eq(0).Text()), " ", "_"))
					value := strings.TrimSpace(cells.Eq(1).Text())
					examResultInfo[key] = value
				}
			})
			// tableData = append(tableData, examResultInfo)
		})

		sectionData = append(sectionData, map[string]interface{}{
			"sectionName": label,
			"content":     tableData,
		})
	})
	// fmt.Println(sectionData)

	return model
}

func GetExamSujectWiseAnalysis2(doc *goquery.Document) (sectionData []map[string]interface{}) {

	// var sectionData []map[string]interface{}
	doc.Find(".grp-cntnr .section-cntnr").Each(func(i int, section *goquery.Selection) {
		label := strings.Replace(strings.TrimSpace(section.Find(".section-lbl").Text()), "Section : ", "", 1)

		// subjectWiseMatrix := make(map[string]int)
		totalQuesCount := 0
		// correctCount := 0
		// InCorrectCount := 0
		skippedCount := 0
		attempted := 0
		// var tableData []map[string]string
		section.Find("table .menu-tbl").Each(func(j int, table *goquery.Selection) {
			// examResultInfo := make(map[string]string)

			table.Find("tr").Each(func(k int, row *goquery.Selection) {
				cells := row.Find("td")
				if cells.Length() >= 2 {
					key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(cells.Eq(0).Text()), " ", "_"))
					value := strings.TrimSpace(cells.Eq(1).Text())

					switch key {
					case "status_:":
						if value == "Answered" {
							attempted++
						} else {
							skippedCount++
						}
						totalQuesCount++
					}
					// examResultInfo[key] = value
				}
			})

			//tableData = append(tableData, examResultInfo)

		})

		sectionData = append(sectionData, map[string]interface{}{
			"sectionName":    label,
			"totalQuesCount": totalQuesCount,
			"skippedCount":   skippedCount,
			"attempted":      attempted,
			// "matrix":         subjectWiseMatrix,
			// "content":     tableData,
		})
	})
	// fmt.Println(sectionData)

	return sectionData
}

func GetExamSujectWiseAnalysis3(doc *goquery.Document) (model []SectionAnalysis, sections []Section) {
	// Find each section
	doc.Find(".section-cntnr").Each(func(i int, sectionSel *goquery.Selection) {
		sectionName := sectionSel.Find(".section-lbl .bold").Text()
		var questions []Question
		section := SectionAnalysis{SectionName: sectionName}
		// Find each question in the section
		sectionSel.Find(".question-pnl").Each(func(j int, questionSel *goquery.Selection) {
			questionID := questionSel.Find(".menu-tbl tr:nth-child(1) td:nth-child(2)").Text()
			// status := questionSel.Find(".menu-tbl tr:nth-child(2) td:nth-child(2)").Text()
			status := questionSel.Find(".menu-tbl tr:contains('Status :') td:nth-child(2)").Text()
			// answer := questionSel.Find(".menu-tbl tr:nth-child(3) td:nth-child(2)").Text()
			answer := questionSel.Find(".menu-tbl tr:contains('Chosen Option :') td:nth-child(2)").Text()

			correctAnswerText := questionSel.Find(".rightAns").Text()
			correctOptionNumber := strings.Split(correctAnswerText, ".")[0]
			correctOptionNumber = strings.TrimSpace(correctOptionNumber)
			isCorrect := correctOptionNumber == answer

			switch status {
			case STATUS_ANSWERED:
				section.Attempted++
				if isCorrect {
					section.Correct++
				} else {
					section.InCorrect++
				}
			case STATUS_ATTEMPTED_AND_MARKFORREVIEW:
				section.MarkForReview++

				section.Attempted++
				if isCorrect {
					section.Correct++
				} else {
					section.InCorrect++
				}
			case STATUS_NOT_ANSWERED:
				section.UnAttempted++
			case STATUS_NOT_ATTEMPTED_AND_MARKFORREVIEW:
				section.MarkForReview++
			}
			section.TotalQues++
			//fmt.Printf("  QuestionID: %s, Status: %s, Answer: %s, IsCorrect: %v\n", questionID, status, answer, isCorrect)
			questions = append(questions, Question{
				QuestionID: questionID,
				Status:     status,
				Answer:     answer,
				IsCorrect:  isCorrect,
			})
		})

		model = append(model, section)
		sections = append(sections, Section{
			Name:      sectionName,
			Questions: questions,
		})
	})
	return model, sections
}

func NewFuncToExtarct(doc *goquery.Document) {
	var sections []Section

	// Find each section
	doc.Find(".section-cntnr").Each(func(i int, sectionSel *goquery.Selection) {
		sectionName := sectionSel.Find(".section-lbl .bold").Text()
		var questions []Question

		// Find each question in the section
		sectionSel.Find(".question-pnl").Each(func(j int, questionSel *goquery.Selection) {
			questionID := questionSel.Find(".menu-tbl tr:nth-child(1) td:nth-child(2)").Text()
			status := questionSel.Find(".menu-tbl tr:nth-child(2) td:nth-child(2)").Text()
			answer := questionSel.Find(".menu-tbl tr:nth-child(3) td:nth-child(2)").Text()
			// isCorrect := questionSel.Find(".rightAns").Length() > 0
			// correctVal := questionSel.Find(".rightAns")

			// Extract the correct answer text (e.g., "3. Abc")
			correctAnswerText := questionSel.Find(".rightAns").Text()
			// Parse the correct option number (e.g., "3")
			correctOptionNumber := strings.Split(correctAnswerText, ".")[0]
			correctOptionNumber = strings.TrimSpace(correctOptionNumber)

			// Compare the correct option number with the selected answer
			isCorrect := correctOptionNumber == answer

			// fmt.Printf("  QuestionID: %s, Status: %s, Answer: %s,correctVal: %s, IsCorrect: %v\n", questionID, status, answer, correctVal.Text(), isCorrect)
			// fmt.Println("questionID:" + questionID + " correctVal: " + correctVal.Text())
			questions = append(questions, Question{
				QuestionID: questionID,
				Status:     status,
				Answer:     answer,
				IsCorrect:  isCorrect,
			})
		})

		sections = append(sections, Section{
			Name:      sectionName,
			Questions: questions,
		})
	})

	//Print the extracted data
	for _, section := range sections {
		fmt.Printf("Section: %s\n", section.Name)
		for _, question := range section.Questions {
			fmt.Printf("  QuestionID: %s, Status: %s, Answer: %s, IsCorrect: %v\n", question.QuestionID, question.Status, question.Answer, question.IsCorrect)
		}
	}
}

type StudentExamInfo struct {
	RollNumber    string `json:"rollNumber"`
	CandidateName string `json:"candidateName"`
	VenueName     string `json:"venueName"`
	ExamDate      string `json:"examDate"`
	ExamTime      string `json:"examTime"`
	Subject       string `json:"subject"`
}

type SectionAnalysis struct {
	SectionName   string  `json:"sectionName"`
	Attempted     int     `json:"attempted"`
	UnAttempted   int     `json:"unAttempted"`
	Correct       int     `json:"correct"`
	InCorrect     int     `json:"incorrect"`
	TotalQues     int     `json:"totalQues"`
	MarkForReview int     `json:"markReviewCount"`
	Marks         float32 `json:"marks"`
}
type ExamResult struct {
	ExamInfo        StudentExamInfo   `json:"examInfo"`
	SubjectAnalysis []SectionAnalysis `json:"subjectAnalysis"`
}

type Question struct {
	QuestionID string `json:"questionId"`
	Status     string `json:"status"`
	Answer     string `json:"answer"`
	IsCorrect  bool   `json:"isCorrect"`
}

type Section struct {
	Name      string     `json:"name"`
	Questions []Question `json:"questions"`
}

const (
	STATUS_ANSWERED                        = "Answered"
	STATUS_NOT_ANSWERED                    = "Not Answered"
	STATUS_ATTEMPTED_AND_MARKFORREVIEW     = "Marked For Review"
	STATUS_NOT_ATTEMPTED_AND_MARKFORREVIEW = "Not Attempted and Marked For Review"
	CHOOSE_OPTION_SKIPPED                  = "--"
)

func test(noCorrectAppttemptedQuestions int) int {
	// fullTestNoOfQuesExpectedForMath := 44.0
	// fullTestNoOfQuesExpectedEnglish := 54.0
	// returnVal := 0
	// defaultMinimumSATScore := 200

	// isMathSection := true
	// // Define the bounds data for SAT
	// bounds := getBoundsBasisOnSection(isMathSection)

	// // Check if the number of questions exists in the map
	// if bound, exists := bounds[noCorrectAppttemptedQuestions]; exists {
	// 	average := (bound.Lower + bound.Upper) / 2
	// 	// Round to the nearest 10
	// 	returnVal = roundToNext10(average)
	// 	return returnVal
	// }
	// returnVal = defaultMinimumSATScore
	// println("returnVal: " + string(returnVal))
	noOfCorrQues := int(19) * 2 // to calculate overall section basis * 2 bcoz we have 2 modules in 1 section
	noOfQues := 22.0 * 2        // to calculate overall section basis * 2 bcoz we have 2 modules in 1 section
	// SatScore = int(satScore / 2) // again / 2 to get overall

	SatScore := GetSATScore(noOfCorrQues, noOfQues, "Math 1", "", false)
	SatScore = roundToNext10(int(SatScore / 2)) // again / 2 to get overall
	println("returnVal: ", SatScore)

	round := roundToNext10(20)
	round = roundToNext10(45)
	println("returnVal: ", round)
	return SatScore
}

type Bounds struct {
	Lower int
	Upper int
}

func GetSATScore(noCorrectAppttemptedQuestions int, totalQuestions float64, sectionName1 string, sectionName2 string, isFullTest bool) int {
	//total No of Ques expected for full test 98
	// fullTestNoOfQuesExpected := 98
	fullTestNoOfQuesExpectedForMath := 44.0
	fullTestNoOfQuesExpectedEnglish := 54.0
	defaultMinimumSATScore := 200
	//defaultMaxSATScore := 800

	if noCorrectAppttemptedQuestions < 0 {
		noCorrectAppttemptedQuestions = 0
	}
	if noCorrectAppttemptedQuestions == 0 || totalQuestions == 0 {
		return defaultMinimumSATScore
	}

	isMathSection := false
	if sectionName1 != "" {
		isMathSection = strings.Contains(strings.ToLower(sectionName1), "math")
	}
	if !isMathSection && sectionName2 != "" {
		isMathSection = strings.Contains(strings.ToLower(sectionName2), "math")
	}

	if isFullTest {
		expectedQues := fullTestNoOfQuesExpectedEnglish
		if isMathSection {
			expectedQues = fullTestNoOfQuesExpectedForMath
		}
		if totalQuestions != expectedQues {
			quesFrac := float64(noCorrectAppttemptedQuestions) / float64(totalQuestions)
			noCorrectAppttemptedQuestions = int(quesFrac * expectedQues)
		}
	}

	// Define the bounds data for SAT
	bounds := getBoundsBasisOnSection(isMathSection)

	// Check if the number of questions exists in the map
	if bound, exists := bounds[noCorrectAppttemptedQuestions]; exists {
		average := (bound.Lower + bound.Upper) / 2
		// Round to the nearest 10
		return roundToNext10(average)
	}
	return defaultMinimumSATScore
}

func roundToNext10(value int) int {
	if value%10 == 0 {
		return value
	}
	if value%10 > 5 {
		return (value/10 + 1) * 10
	}
	return (value / 10) * 10
}

func getBoundsBasisOnSection(isMathSection bool) map[int]Bounds {
	if isMathSection {
		return map[int]Bounds{
			0:  {200, 200},
			1:  {200, 200},
			2:  {200, 200},
			3:  {200, 200},
			4:  {200, 200},
			5:  {200, 200},
			6:  {210, 220},
			7:  {230, 240},
			8:  {240, 250},
			9:  {260, 270},
			10: {270, 280},
			11: {290, 300},
			12: {300, 310},
			13: {320, 330},
			14: {330, 340},
			15: {350, 360},
			16: {360, 370},
			17: {380, 390},
			18: {390, 400},
			19: {410, 420},
			20: {420, 430},
			21: {440, 450},
			22: {450, 460},
			23: {470, 480},
			24: {480, 490},
			25: {500, 510},
			26: {510, 520},
			27: {530, 540},
			28: {540, 550},
			29: {560, 570},
			30: {570, 580},
			31: {590, 600},
			32: {600, 610},
			33: {620, 630},
			34: {630, 640},
			35: {650, 660},
			36: {660, 670},
			37: {680, 690},
			38: {690, 700},
			39: {710, 720},
			40: {720, 730},
			41: {740, 750},
			42: {750, 760},
			43: {770, 780},
			44: {800, 800},
		}
	} else {
		return map[int]Bounds{
			0:  {200, 200},
			1:  {200, 200},
			2:  {200, 200},
			3:  {200, 200},
			4:  {200, 200},
			5:  {200, 200},
			6:  {200, 200},
			7:  {210, 220},
			8:  {220, 230},
			9:  {230, 240},
			10: {240, 250},
			11: {250, 260},
			12: {260, 270},
			13: {280, 290},
			14: {290, 300},
			15: {300, 310},
			16: {310, 320},
			17: {330, 340},
			18: {340, 350},
			19: {350, 360},
			20: {360, 370},
			21: {370, 380},
			22: {390, 400},
			23: {400, 410},
			24: {410, 420},
			25: {420, 430},
			26: {440, 450},
			27: {450, 460},
			28: {460, 470},
			29: {470, 480},
			30: {480, 490},
			31: {500, 510},
			32: {510, 520},
			33: {520, 530},
			34: {530, 540},
			35: {550, 560},
			36: {560, 570},
			37: {570, 580},
			38: {580, 590},
			39: {590, 600},
			40: {610, 620},
			41: {620, 630},
			42: {630, 640},
			43: {640, 650},
			44: {660, 670},
			45: {670, 680},
			46: {680, 690},
			47: {690, 700},
			48: {700, 710},
			49: {720, 730},
			50: {730, 740},
			51: {740, 750},
			52: {750, 760},
			53: {770, 780},
			54: {800, 800},
		}
	}
}

func getBoundsForSectionalBasisOnSection(isMathSection bool) map[int]Bounds {
	if isMathSection {
		return map[int]Bounds{
			0:  {200, 200},
			1:  {200, 200},
			2:  {200, 200},
			3:  {200, 200},
			4:  {200, 200},
			5:  {200, 200},
			6:  {210, 220},
			7:  {220, 230},
			8:  {230, 240},
			9:  {250, 260},
			10: {260, 270},
			11: {270, 280},
			12: {280, 290},
			13: {300, 310},
			14: {310, 320},
			15: {320, 330},
			16: {340, 350},
			17: {350, 360},
			18: {360, 370},
			19: {370, 380},
			20: {380, 390},
			21: {390, 400},
			22: {400, 400}, // Adjusted to be the maximum
		}
	} else {
		return map[int]Bounds{
			0:  {200, 200},
			1:  {200, 200},
			2:  {200, 200},
			3:  {200, 200},
			4:  {200, 200},
			5:  {200, 200},
			6:  {200, 200},
			7:  {210, 220},
			8:  {220, 230},
			9:  {230, 240},
			10: {240, 250},
			11: {250, 260},
			12: {260, 270},
			13: {270, 280},
			14: {280, 290},
			15: {290, 300},
			16: {300, 310},
			17: {310, 320},
			18: {320, 330},
			19: {330, 340},
			20: {340, 350},
			21: {350, 360},
			22: {360, 370},
			23: {370, 380},
			24: {380, 390},
			25: {390, 395},
			26: {395, 400},
			27: {400, 400},
		}
	}
}
