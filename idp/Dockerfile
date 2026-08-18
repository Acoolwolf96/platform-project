FROM golang:1.24-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /src

COPY go.mod .
COPY *.go .
COPY web ./web

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /idp .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /idp /idp

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/idp"]
