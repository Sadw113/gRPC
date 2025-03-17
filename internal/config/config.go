package config

// Общая конфигурация сервиса, тут должны быть все переменные

type AppConfig struct {
	LogLevel string
	GRPC     gRPC
}

type gRPC struct {
	ListenAddress string `envconfig:"PORT" required:"true"`
	Token         string `envconfig:"TOKEN" required:"true"`
}
