package interceptors

import (
	"context"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc/metadata"
)

type grpcCallerOrigin struct{}

type CallerOrigin string

const (
	InternalCallerOrigin = CallerOrigin("internal")
	ExternalCallerOrigin = CallerOrigin("external")
)

type GRPCMetricsExtension struct {
	grpc_prometheus.DefaultExtension
}

func (e *GRPCMetricsExtension) ServerStreamMsgReceivedCounterCustomLabels() []string {
	return []string{"caller_origin"}
}

func (e *GRPCMetricsExtension) ServerStreamMsgReceivedCounterValues(ctx context.Context) []string {
	return []string{GetCallerOriginStr(ctx)}
}

var _ grpc_prometheus.ServerExtension = &GRPCMetricsExtension{}

func SetCallerOrigin(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, grpcCallerOrigin{}, callerOriginFromRequest(ctx, method))
}

func GetCallerOriginStr(ctx context.Context) string {
	v := ctx.Value(grpcCallerOrigin{})
	if v == nil {
		return ""
	}
	return string(v.(CallerOrigin))
}

func callerOriginFromRequest(ctx context.Context, method string) CallerOrigin {
	if method == "/server.Server/Readiness" || method == "/agent.Agent/Connect" {
		return InternalCallerOrigin
	}

	headers, _ := metadata.FromIncomingContext(ctx)

	// if referer is present - the caller is an external one
	if len(headers.Get("grpcgateway-referer")) == 0 {
		return ExternalCallerOrigin
	}

	return InternalCallerOrigin
}
