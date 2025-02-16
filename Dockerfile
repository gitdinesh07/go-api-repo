FROM golang:1.24.0-alpine3.21

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /go_app

EXPOSE 8090

# Run
CMD ["/go_app"]