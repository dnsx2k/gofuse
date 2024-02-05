FROM golang:1.21.6

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .
RUN go build -o main ./cmd

EXPOSE 8080

CMD ["./main"]