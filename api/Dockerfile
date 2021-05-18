FROM alpine:latest as ca-certs

RUN apk add -U --no-cache ca-certificates

FROM golang:1.16-buster as builder

WORKDIR /usr/src/app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -v -o api main.go

FROM scratch
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/src/app/api /app/bin/api
EXPOSE 8080
ENTRYPOINT [ "/app/bin/api" ]