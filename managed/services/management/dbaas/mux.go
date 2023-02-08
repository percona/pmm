package dbaas

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
)

type KubeExposer struct {
	l           *logrus.Entry
	kubeStorage *KubeStorage
	router      *gin.Engine
}

func NewKubeExposer(db *reform.DB, mux *http.ServeMux) *KubeExposer {
	l := logrus.WithField("component", "kube_exposer")
	e := &KubeExposer{
		kubeStorage: NewKubeStorage(db),
		router:      gin.Default(),
		l:           l,
	}
	e.initRoutes(mux)
	return e
}
func (s *KubeExposer) initRoutes(mux *http.ServeMux) {
	s.router.Any("/v1/kubernetes/:name/*proxyPath", s.proxyK8s)
	//s.router.Any((gin.WrapF(mux.ServeHTTP))
}
func (s *KubeExposer) proxyK8s(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		return
	}
	kubeClient, err := s.kubeStorage.GetOrSetClient(name)
	if err != nil {
		s.l.Error(err)
		return
	}
	transport, err := kubeClient.GetTransport()
	if err != nil {
		s.l.Error(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Host:   strings.TrimPrefix(kubeClient.GetHost(), "https://"),
		Scheme: "https",
	})
	reverseProxy.Transport = transport
	req := c.Request
	req.URL.Path = strings.TrimLeft(req.URL.Path, fmt.Sprintf("/v1/kubernetes/%s", name))
	reverseProxy.ServeHTTP(c.Writer, req)
}
func (s *KubeExposer) HandleKubernetes(w http.ResponseWriter, req *http.Request) {
	name := strings.Split(strings.TrimPrefix(req.URL.Path, "/v1/kubernetes/"), "/")[0]
	if name == "" {
		return
	}
	kubeClient, err := s.kubeStorage.GetOrSetClient(name)
	if err != nil {
		s.l.Error(err)
		return
	}
	transport, err := kubeClient.GetTransport()
	if err != nil {
		s.l.Error(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Host:   strings.TrimPrefix(kubeClient.GetHost(), "https://"),
		Scheme: "https",
	})
	reverseProxy.Transport = transport
	req.URL.Path = strings.TrimLeft(req.URL.Path, fmt.Sprintf("/v1/kubernetes/%s", name))
	reverseProxy.ServeHTTP(w, req)
}
func (s *KubeExposer) Handler() *gin.Engine {
	return s.router
}
