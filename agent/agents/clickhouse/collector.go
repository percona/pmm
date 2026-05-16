package clickhouse

import (
	"database/sql"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// Collector coleta métricas do ClickHouse.
type Collector struct {
	queryCount    *prometheus.Desc
	scrapeSeconds *prometheus.Desc
	client        *sql.DB
}

// NewCollector cria um novo Collector para ClickHouse usando o DSN.
func NewCollector(dsn string) (*Collector, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
	}
	// Verifica a conexão
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Collector{
		queryCount: prometheus.NewDesc(
			"clickhouse_query_count",
			"Número de queries executadas no ClickHouse (último minuto)",
			nil, nil,
		),
		scrapeSeconds: prometheus.NewDesc(
			"clickhouse_scrape_duration_seconds",
			"Tempo gasto para coletar métricas do ClickHouse",
			nil, nil,
		),
		client: db,
	}, nil
}

// Describe envia os descritores das métricas.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.queryCount
	ch <- c.scrapeSeconds
}

// Collect executa a coleta das métricas.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	// Exemplo de consulta: contar queries dos últimos 1 minuto na tabela system.query_log.
	// Ajuste essa consulta conforme as necessidades e versão do ClickHouse.
	var count int
	err := c.client.QueryRow("SELECT count(*) FROM system.query_log WHERE event_time >= now() - interval 1 minute").Scan(&count)
	if err != nil {
		log.Printf("Erro ao coletar métricas do ClickHouse: %v", err)
		return
	}
	duration := time.Since(start).Seconds()

	ch <- prometheus.MustNewConstMetric(c.queryCount, prometheus.GaugeValue, float64(count))
	ch <- prometheus.MustNewConstMetric(c.scrapeSeconds, prometheus.GaugeValue, duration)
}
