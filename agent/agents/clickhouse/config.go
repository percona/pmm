// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
