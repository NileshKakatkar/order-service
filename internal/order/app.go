package order

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/nilesh/order-service/internal/order/rpc"
	orderevent "github.com/nilesh/shopcore-events/order"
	"github.com/segmentio/kafka-go"
	"net/http"
	"strings"
	"time"
)

func NewApp(cfg Config) (*fiber.App, error) {
	db, err := openDatabase(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	app := fiber.New(fiber.Config{AppName: "ShopCore Order Service", BodyLimit: 1 << 20})
	app.Use(recover.New())
	rpcClients, err := rpc.NewClients(cfg.UserGRPCAddress, cfg.ProductGRPCAddress)
	if err != nil {
		db.Close()
		return nil, err
	}
	handler := &Handler{DB: db, ProductURL: cfg.ProductServiceURL, ServiceKey: cfg.InternalServiceKey, HTTP: &http.Client{Timeout: 8 * time.Second}, RPC: rpcClients}
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok", "service": "order"}) })
	secured := app.Group("/api/v1", authenticate(cfg.PublicKeyPath))
	secured.Post("/orders", handler.Create)
	secured.Get("/orders", handler.List)
	secured.Get("/orders/:id", handler.Get)
	writer := &kafka.Writer{Addr: kafka.TCP(splitBrokers(cfg.KafkaBrokers)...), Topic: orderevent.PlacedTopic, Balancer: &kafka.LeastBytes{}, Async: false}
	dispatcher := &OutboxDispatcher{DB: db, Writer: writer}
	ctx := context.Background()
	go dispatcher.Run(ctx)
	return app, nil
}

func splitBrokers(value string) []string {
	parts := []string{}
	for _, part := range strings.Split(value, ",") {
		if p := strings.TrimSpace(part); p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
