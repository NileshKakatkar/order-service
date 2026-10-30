# Order Service

Fiber service for creating and listing orders. It retrieves user and product details through gRPC, reserves inventory through the Product Service HTTP API, and writes `order.placed` events to a PostgreSQL outbox. The outbox dispatcher publishes those events to Kafka.

## Run

```sh
go mod download
go run ./cmd/order
```

Default port: `8083`. Keep `shared/grpc` and `shared/events` beside this service. Both are separate Go modules referenced locally.
