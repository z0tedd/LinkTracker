package config

import "time"

type Config struct {
	BotToken             string        `env:"TELEGRAM_BOT_TOKEN"`
	AccessType           string        `env:"DB_ACCESS_TYPE" envDefault:"SQL"`                                           // SQL or ORM
	DBURL                string        `env:"DB_URL" envDefault:"jdbc:postgresql://postgres_container:5432/postgres_db"` // SQL or ORM
	Crontab              string        `env:"CRON" envDefault:"0/10 * * * *"`
	MessageTransportType string        `env:"MESSAGE_TRANSPORT_TYPE" envDefault:"http"`
	StateManagerType     string        `env:"STATE_MANAGER_TYPE" envDefault:"Redis"`
	KafkaAddresses       string        `env:"KAFKA_ADDRESSES" envDefault:"localhost:9092"`
	BotBaseURL           string        `env:"BOT_BASE_URL" envDefault:"http://localhost:8081"`
	ScrapperGroupID      string        `env:"SCRAPPER_GROUP_ID" envDefault:"scrapper_consumer"`
	ScrapperHTTPAddress  string        `env:"SCRAPPER_HTTP_ADDRESS" envDefault:"http://localhost:8080"`
	BotGroupID           string        `env:"BOT_GROUP_ID" envDefault:"bot_consumer"`
	KafkaTopic           string        `env:"KAFKA_TOPIC" envDefault:"messages"`
	RedisURL             string        `env:"REDIS_URL"`
	Timeout              time.Duration `env:"TIMEOUT" envDefault:"10s"`            // Timeout duration
	RateLimit            uint          `env:"RATE_LIMIT" envDefault:"15"`          // Rate limit per second
	Burst                int           `env:"BURST" envDefault:"5"`                // Burst capacity for rate limiting
	ExpiresIn            time.Duration `env:"EXPIRES_IN" envDefault:"5s"`          // Burst capacity for rate limiting
	RetryCount           uint          `env:"RETRY_COUNT" envDefault:"15"`         // Number of retry attempts
	InitialRetryDelay    time.Duration `env:"INITIAL_RETRY_DELAY" envDefault:"1s"` // Initial delay for retries
}
