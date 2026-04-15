FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/staff-service ./services/staff-service/cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && adduser -D -H app

WORKDIR /app
COPY --from=builder /out/staff-service /app/staff-service

EXPOSE 50052

USER app
CMD ["/app/staff-service"]

