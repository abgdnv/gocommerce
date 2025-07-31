FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY user_service/go.mod user_service/go.sum ./user_service/
COPY pkg/ ./pkg/

WORKDIR /app/user_service
RUN go mod download

WORKDIR /app
COPY user_service/ ./user_service/

WORKDIR /app/user_service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o app github.com/abgdnv/gocommerce/user_service/cmd/

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /app/user_service/app .

RUN adduser -D -g '' appuser && chown appuser:appuser /app/app
USER appuser

EXPOSE 50051

CMD ["./app"]
