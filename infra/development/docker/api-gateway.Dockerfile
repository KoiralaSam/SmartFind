FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api-gateway ./services/api-gateway

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && adduser -D -H app

WORKDIR /app
COPY --from=builder /out/api-gateway /app/api-gateway

EXPOSE 8081

USER app
CMD ["/app/api-gateway"]

