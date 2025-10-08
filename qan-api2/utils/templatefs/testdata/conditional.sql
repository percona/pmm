-- Template with conditional logic
CREATE TABLE {{.TableName}} (
    id BIGINT PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL
);

{{if .AddIndexes}}
CREATE INDEX {{.IndexName}} ON {{.TableName}} ({{.ColumnName}});
{{end}}
