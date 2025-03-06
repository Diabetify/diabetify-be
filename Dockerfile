FROM golang:1.23-alpine

ENV TZ='Asia/Jakarta'

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

WORKDIR /app/cmd
CMD ["go", "run", "main.go"]