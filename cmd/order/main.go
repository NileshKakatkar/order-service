package main

import (
	"log"

	"github.com/nilesh/order-service/internal/order"
)

func main() {
	cfg := order.LoadConfig()
	app, err := order.NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
