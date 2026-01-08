package logger

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// PromHTTP is a compatibility wrapper between zap's sugared logger entry
// and Prometheus HTTP logger interface.
type PromHTTP struct {
	L *zap.SugaredLogger
}

// Println prints log message with info level.
func (p *PromHTTP) Println(args ...interface{}) { p.L.Info(args...) }

// Check interfaces.
var _ promhttp.Logger = (*PromHTTP)(nil)
