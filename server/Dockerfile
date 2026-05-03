# ── dev target: air live-reload (used via docker-compose.override.yml) ────────
FROM golang:1.25-alpine AS dev

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

EXPOSE 8080
CMD ["air"]

# ── production build ───────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sally-server ./cmd/sally-server

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /out/sally-server /usr/local/bin/sally-server
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/sally-server"]
