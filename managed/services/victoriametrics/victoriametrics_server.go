package victoriametrics

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type Server struct {
	l               *logrus.Entry
	victoriaMetrics *Service
}

func NewServer(victoriaMetrics *Service) *Server {
	return &Server{
		victoriaMetrics: victoriaMetrics,
		l:               logrus.WithField("component", "victoriametrics_server"),
	}
}

// ServeHTTP serves internal location /scrape_configs for requests from victoria metrics.
func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	config, err := s.victoriaMetrics.generateConfig()
	if err != nil {
		s.l.Errorf("couldn't generate config: %q", err)
		rw.WriteHeader(500)
		return
	}

	_, err = rw.Write(config)
	if err != nil {
		s.l.Errorf("couldn't write response: %q", err)
		rw.WriteHeader(500)
		return
	}

}
