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
	grpc_prometheus.NullExtension
}

func (GRPCMetricsExtension) ServerHandledCounterCustomLabels() []string {
	return []string{"caller_origin"}
}

func (GRPCMetricsExtension) ServerHandledCounterPreRegisterValues() [][]string {
	return [][]string{
		{string(InternalCallerOrigin)},
		{string(ExternalCallerOrigin)},
	}
}

func (GRPCMetricsExtension) ServerHandledCounterValues(ctx context.Context) []string {
	return []string{GetCallerOriginStr(ctx)}
}

func SetCallerOrigin(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, grpcCallerOrigin{}, callerOriginFromRequest(ctx, method))
}

func GetCallerOriginStr(ctx context.Context) string {
	value, ok := ctx.Value(grpcCallerOrigin{}).(CallerOrigin)
	if ok {
		return string(value)
	}

	return ""
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
