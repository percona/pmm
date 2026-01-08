package logger

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"go.uber.org/zap"
)

// HTTPOption is an option for http.RoundTripper returned by HTTP constructor.
type HTTPOption func(*roundTripper)

// LogFullRequest enable/disables logging of request/response body and headers.
// Enable only for local development!
func LogFullRequest() HTTPOption {
	return func(rt *roundTripper) {
		rt.logFullRequest = true
	}
}

// HTTP returns http.RoundTripper with request/response logger.
func HTTP(rt http.RoundTripper, loggerName string, opts ...HTTPOption) http.RoundTripper {
	out := &roundTripper{
		rt:         rt,
		loggerName: loggerName,
	}

	for _, opt := range opts {
		opt(out)
	}

	return out
}

type roundTripper struct {
	rt         http.RoundTripper
	loggerName string

	// log all request/response headers and body
	// default is false to prevent accidental private info logging
	logFullRequest bool
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rl := GetLoggerFromContext(req.Context())
	if rt.loggerName != "" {
		rl = rl.Named(rt.loggerName)
	}

	if rl.Core().Enabled(zap.DebugLevel) && rt.logFullRequest {
		b, _ := httputil.DumpRequestOut(req, true)
		if len(b) != 0 {
			rl.Debug(fmt.Sprintf("Sending request:\n%s.", b))
		}
	} else {
		rl.Info(fmt.Sprintf("Sending request to host=%s.", req.URL.Host))
	}

	resp, err := rt.rt.RoundTrip(req)

	if err != nil { //nolint: nestif
		rl.Error("Received error", zap.Error(err))
	} else if resp != nil {
		if rl.Core().Enabled(zap.DebugLevel) && rt.logFullRequest {
			b, _ := httputil.DumpResponse(resp, true)
			if len(b) != 0 {
				rl.Debug(fmt.Sprintf("Received response:\n%s", b))
			}
		} else {
			rl.Info("Received response: " + resp.Status)
		}
	}
	return resp, err
}
