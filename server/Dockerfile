FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sally-server ./cmd/sally-server

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /out/sally-server /usr/local/bin/sally-server

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/sally-server"]
