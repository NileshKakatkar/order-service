package order

import "os"

type Config struct{ Port, DatabaseURL, PublicKeyPath, ProductServiceURL, UserGRPCAddress, ProductGRPCAddress, KafkaBrokers, InternalServiceKey string }

func LoadConfig() Config {
	get := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return Config{Port: get("ORDER_SERVICE_PORT", "8083"), DatabaseURL: "postgres://" + get("POSTGRES_USER", "shopcore") + ":" + get("POSTGRES_PASSWORD", "shopcore") + "@" + get("POSTGRES_HOST", "localhost") + ":" + get("POSTGRES_PORT", "5432") + "/shopcore_orders?sslmode=" + get("POSTGRES_SSLMODE", "disable"), PublicKeyPath: get("JWT_PUBLIC_KEY_PATH", "./secrets/public.pem"), ProductServiceURL: get("PRODUCT_SERVICE_URL", "http://localhost:8082"), UserGRPCAddress: get("USER_GRPC_ADDRESS", "localhost:50051"), ProductGRPCAddress: get("PRODUCT_GRPC_ADDRESS", "localhost:50052"), KafkaBrokers: get("KAFKA_BROKERS", "localhost:9092"), InternalServiceKey: get("INTERNAL_SERVICE_KEY", "")}
}
