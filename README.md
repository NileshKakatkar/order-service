# Order Service

Fiber service for creating and listing orders. It calls the Product Service to reserve inventory and publishes order events through a PostgreSQL outbox to Kafka.

## Run

```sh
go mod download
go run ./cmd/order
```

Default port: `8083`.
