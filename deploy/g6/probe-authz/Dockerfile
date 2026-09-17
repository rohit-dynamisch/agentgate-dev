FROM golang:1.24-alpine AS builder

ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /probe-authz .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /probe-authz /app/probe-authz
EXPOSE 9001 9002
ENTRYPOINT ["/app/probe-authz"]
