# Build stage only — Go compiler never exists in the final image
FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod .
COPY main.go .
COPY web ./web

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /idp .

# Tiny runtime image
FROM scratch

COPY --from=builder /idp /idp

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/idp"]
