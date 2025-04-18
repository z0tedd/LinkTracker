package config

type Config struct {
	BotToken             string `env:"TELEGRAM_BOT_TOKEN"`
	AccessType           string `env:"DB_ACCESS_TYPE" envDefault:"SQL"`                                           // SQL or ORM
	DBURL                string `env:"DB_URL" envDefault:"jdbc:postgresql://postgres_container:5432/postgres_db"` // SQL or ORM
	Crontab              string `env:"CRON" envDefault:"0/10 * * * *"`
	MessageTransportType string `env:"MESSAGE_TRANSPORT_TYPE" envDefault:"http"`
	KafkaAddresses       string `env:"KAFKA_ADDRESSES" envDefault:"localhost:9092"`
	BotBaseURL           string `env:"BOT_BASE_URL" envDefault:"http://localhost:8081"`
	ScrapperGroupID      string `env:"SCRAPPER_GROUP_ID" envDefault:"scrapper_consumer"`
	BotGroupID           string `env:"BOT_GROUP_ID" envDefault:"bot_consumer"`
	KafkaTopic           string `env:"KAFKA_TOPIC" envDefault:"messages"`
}
