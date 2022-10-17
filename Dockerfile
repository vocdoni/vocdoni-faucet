FROM golang:1.18-alpine3.15 AS builder

WORKDIR /src
COPY . .
RUN apk update && apk add build-base
RUN go build -o=main -ldflags="-s -w" ./cmd/

FROM alpine:3.13

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
COPY --from=builder /src/main ./
ENTRYPOINT ["/app/main"]