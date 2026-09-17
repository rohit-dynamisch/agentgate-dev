FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /probe-mcp .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /probe-mcp /app/probe-mcp
EXPOSE 9100 9101
ENTRYPOINT ["/app/probe-mcp"]
