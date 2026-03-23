FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/auth-service ./cmd/auth-service

FROM alpine:3.21

WORKDIR /app

COPY --from=build /out/auth-service /usr/local/bin/auth-service

EXPOSE 50052

ENTRYPOINT ["/usr/local/bin/auth-service"]
