package config

type Config struct {
	BotToken   string `env:"TELEGRAM_BOT_TOKEN"`
	AccessType string `env:"DB_ACCESS_TYPE" envDefault:"SQL"`                                           // SQL or ORM
	DBURL      string `env:"DB_URL" envDefault:"jdbc:postgresql://postgres_container:5432/postgres_db"` // SQL or ORM
}
