FROM golang:1.23-alpine AS builder
WORKDIR /src/order-service
COPY contracts/ /src/contracts/
COPY order-service/go.mod ./
RUN go mod download
COPY order-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/order

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/service /app/service
USER app
EXPOSE 8083
ENTRYPOINT ["/app/service"]
