package clickhouse

import "os"

// Config contém as configurações para conectar ao ClickHouse.
type Config struct {
	// DSN no formato: tcp://<host>:<port>?username=<user>&password=<pass>&database=<db>
	DSN string
	// Porta de scrape que o exportador usará (opcional, para registrar no pmm-agent)
	ScrapePort string
}

// LoadConfig lê as configurações a partir das variáveis de ambiente.
// Variáveis esperadas: CLICKHOUSE_DSN e CLICKHOUSE_SCRAPE_PORT.
func LoadConfig() *Config {
	dsn := os.Getenv("CLICKHOUSE_DSN")
	if dsn == "" {
		// Valor padrão – ajuste conforme necessário
		dsn = "tcp://localhost:9000?username=default&password=&database=default"
	}
	scrapePort := os.Getenv("CLICKHOUSE_SCRAPE_PORT")
	if scrapePort == "" {
		scrapePort = "9100"
	}
	return &Config{
		DSN:        dsn,
		ScrapePort: scrapePort,
	}
}
