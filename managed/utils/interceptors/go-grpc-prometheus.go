// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package interceptors contains gRPC wrappers for logging and Prometheus metrics.
// Beware: this code is copied from Percona-Lab/go-grpc-prometheus to get rid of that fork.
// Everything needed is contained in this file. There is a ticket (PMM-14659) related to
// upgrading to go-grpc-middleware. We need to upgrade in the near future because the upstream
// go-grpc-prometheus repository is now archived. After upgrade, this file should be removed/updated.
// Lint is disabled for this file because the code is expected to be removed in the near future.
// Remove excluded lint rule for managed/utils/interceptors/go-grpc-prometheus.go in .golangci.yml after the file is removed.
package interceptors

import (
	"context"
	"strings"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCMetricsExtensionEmpty struct {
	DefaultExtension
}

var _ ServerExtension = &GRPCMetricsExtensionEmpty{}

type ServerExtension interface {
	ServerHandledCounterCustomLabels() []string
	ServerHandledCounterValues(ctx context.Context) []string

	ServerStreamMsgReceivedCounterCustomLabels() []string
	ServerStreamMsgReceivedCounterValues(ctx context.Context) []string

	ServerStreamMsgSentCounterCustomLabels() []string
	ServerStreamMsgSentCounterValues(ctx context.Context) []string

	ServerHandledHistogramCustomLabels() []string
	ServerHandledHistogramValues(ctx context.Context) []string
}

type DefaultExtension struct{}

func (DefaultExtension) ServerHandledCounterCustomLabels() []string {
	return nil
}

func (DefaultExtension) ServerHandledCounterValues(context.Context) []string {
	return nil
}

func (DefaultExtension) ServerStreamMsgReceivedCounterCustomLabels() []string {
	return nil
}

func (DefaultExtension) ServerStreamMsgReceivedCounterValues(context.Context) []string {
	return nil
}

func (DefaultExtension) ServerStreamMsgSentCounterCustomLabels() []string {
	return nil
}

func (DefaultExtension) ServerStreamMsgSentCounterValues(context.Context) []string {
	return nil
}

func (DefaultExtension) ServerHandledHistogramCustomLabels() []string {
	return nil
}

func (DefaultExtension) ServerHandledHistogramValues(context.Context) []string {
	return nil
}

// A CounterOption lets you add options to Counter metrics using With* funcs.
type CounterOption func(*prom.CounterOpts)

type counterOptions []CounterOption

func (co counterOptions) apply(o prom.CounterOpts) prom.CounterOpts {
	for _, f := range co {
		f(&o)
	}
	return o
}

// WithConstLabels allows you to add ConstLabels to Counter metrics.
func WithConstLabels(labels prom.Labels) CounterOption {
	return func(o *prom.CounterOpts) {
		o.ConstLabels = labels
	}
}

// A HistogramOption lets you add options to Histogram metrics using With*
// funcs.
type HistogramOption func(*prom.HistogramOpts)

// WithHistogramBuckets allows you to specify custom bucket ranges for histograms if EnableHandlingTimeHistogram is on.
func WithHistogramBuckets(buckets []float64) HistogramOption {
	return func(o *prom.HistogramOpts) { o.Buckets = buckets }
}

// WithHistogramConstLabels allows you to add custom ConstLabels to
// histograms metrics.
func WithHistogramConstLabels(labels prom.Labels) HistogramOption {
	return func(o *prom.HistogramOpts) {
		o.ConstLabels = labels
	}
}

// ServerMetrics represents a collection of metrics to be registered on a
// Prometheus metrics registry for a gRPC server.
type ServerMetrics struct {
	extension                      ServerExtension
	serverStartedCounter           *prom.CounterVec
	serverHandledCounter           *prom.CounterVec
	serverStreamMsgReceivedCounter *prom.CounterVec
	serverStreamMsgSentCounter     *prom.CounterVec
	serverHandledHistogramEnabled  bool
	serverHandledHistogramOpts     prom.HistogramOpts
	serverHandledHistogram         *prom.HistogramVec
}

// NewServerMetrics returns a ServerMetrics object. Use a new instance of
// ServerMetrics when not using the default Prometheus metrics registry, for
// example when wanting to control which metrics are added to a registry as
// opposed to automatically adding metrics via init functions.

func NewServerMetricsWithExtension(extension ServerExtension, counterOpts ...CounterOption) *ServerMetrics {
	opts := counterOptions(counterOpts)
	return &ServerMetrics{
		extension: extension,
		serverStartedCounter: prom.NewCounterVec(
			opts.apply(prom.CounterOpts{
				Name: "grpc_server_started_total",
				Help: "Total number of RPCs started on the server.",
			}), []string{"grpc_type", "grpc_service", "grpc_method"},
		),
		serverHandledCounter: prom.NewCounterVec(
			opts.apply(prom.CounterOpts{
				Name: "grpc_server_handled_total",
				Help: "Total number of RPCs completed on the server, regardless of success or failure.",
			}), append(extension.ServerHandledCounterCustomLabels(), "grpc_type", "grpc_service", "grpc_method", "grpc_code"),
		),
		serverStreamMsgReceivedCounter: prom.NewCounterVec(
			opts.apply(prom.CounterOpts{
				Name: "grpc_server_msg_received_total",
				Help: "Total number of RPC stream messages received on the server.",
			}), append(extension.ServerStreamMsgReceivedCounterCustomLabels(), "grpc_type", "grpc_service", "grpc_method"),
		),
		serverStreamMsgSentCounter: prom.NewCounterVec(
			opts.apply(prom.CounterOpts{
				Name: "grpc_server_msg_sent_total",
				Help: "Total number of gRPC stream messages sent by the server.",
			}), append(extension.ServerStreamMsgSentCounterCustomLabels(), "grpc_type", "grpc_service", "grpc_method"),
		),
		serverHandledHistogramEnabled: false,
		serverHandledHistogramOpts: prom.HistogramOpts{
			Name:    "grpc_server_handling_seconds",
			Help:    "Histogram of response latency (seconds) of gRPC that had been application-level handled by the server.",
			Buckets: prom.DefBuckets,
		},
		serverHandledHistogram: nil,
	}
}

// EnableHandlingTimeHistogram enables histograms being registered when
// registering the ServerMetrics on a Prometheus registry. Histograms can be
// expensive on Prometheus servers. It takes options to configure histogram
// options such as the defined buckets.
func (m *ServerMetrics) EnableHandlingTimeHistogram(opts ...HistogramOption) {
	for _, o := range opts {
		o(&m.serverHandledHistogramOpts)
	}
	if !m.serverHandledHistogramEnabled {
		m.serverHandledHistogram = prom.NewHistogramVec(
			m.serverHandledHistogramOpts,
			[]string{"grpc_type", "grpc_service", "grpc_method"},
		)
	}
	m.serverHandledHistogramEnabled = true
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (m *ServerMetrics) Describe(ch chan<- *prom.Desc) {
	m.serverStartedCounter.Describe(ch)
	m.serverHandledCounter.Describe(ch)
	m.serverStreamMsgReceivedCounter.Describe(ch)
	m.serverStreamMsgSentCounter.Describe(ch)
	if m.serverHandledHistogramEnabled {
		m.serverHandledHistogram.Describe(ch)
	}
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (m *ServerMetrics) Collect(ch chan<- prom.Metric) {
	m.serverStartedCounter.Collect(ch)
	m.serverHandledCounter.Collect(ch)
	m.serverStreamMsgReceivedCounter.Collect(ch)
	m.serverStreamMsgSentCounter.Collect(ch)
	if m.serverHandledHistogramEnabled {
		m.serverHandledHistogram.Collect(ch)
	}
}

// UnaryServerInterceptor is a gRPC server-side interceptor that provides Prometheus monitoring for Unary RPCs.
func (m *ServerMetrics) UnaryServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		monitor := newServerReporter(m, Unary, info.FullMethod)
		monitor.ReceivedMessage(ctx)
		resp, err := handler(ctx, req)
		st, _ := status.FromError(err)
		monitor.Handled(ctx, st.Code())
		if err == nil {
			monitor.SentMessage(ctx)
		}
		return resp, err
	}
}

// StreamServerInterceptor is a gRPC server-side interceptor that provides Prometheus monitoring for Streaming RPCs.
func (m *ServerMetrics) StreamServerInterceptor() func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		monitor := newServerReporter(m, streamRPCType(info), info.FullMethod)
		err := handler(srv, &monitoredServerStream{ss, monitor})
		st, _ := status.FromError(err)
		monitor.Handled(ss.Context(), st.Code())
		return err
	}
}

// InitializeMetrics initializes all metrics, with their appropriate null
// value, for all gRPC methods registered on a gRPC server. This is useful, to
// ensure that all metrics exist when collecting and querying.
func (m *ServerMetrics) InitializeMetrics(server *grpc.Server) {
	serviceInfo := server.GetServiceInfo()
	for serviceName, info := range serviceInfo {
		for _, mInfo := range info.Methods {
			preRegisterMethod(m, serviceName, &mInfo)
		}
	}
}

func streamRPCType(info *grpc.StreamServerInfo) grpcType {
	if info.IsClientStream && !info.IsServerStream {
		return ClientStream
	} else if !info.IsClientStream && info.IsServerStream {
		return ServerStream
	}
	return BidiStream
}

// monitoredStream wraps grpc.ServerStream allowing each Sent/Recv of message to increment counters.
type monitoredServerStream struct {
	grpc.ServerStream
	monitor *serverReporter
}

func (s *monitoredServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.monitor.SentMessage(s.ServerStream.Context())
	}
	return err
}

func (s *monitoredServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.monitor.ReceivedMessage(s.ServerStream.Context())
	}
	return err
}

// preRegisterMethod is invoked on Register of a Server, allowing all gRPC services labels to be pre-populated.
func preRegisterMethod(metrics *ServerMetrics, serviceName string, mInfo *grpc.MethodInfo) {
	methodName := mInfo.Name
	methodType := string(typeFromMethodInfo(mInfo))
	// These are just references (no increments), as just referencing will create the labels but not set values.
	metrics.serverStartedCounter.GetMetricWithLabelValues(methodType, serviceName, methodName)
	metrics.serverStreamMsgReceivedCounter.GetMetricWithLabelValues(methodType, serviceName, methodName)
	metrics.serverStreamMsgSentCounter.GetMetricWithLabelValues(methodType, serviceName, methodName)
	if metrics.serverHandledHistogramEnabled {
		metrics.serverHandledHistogram.GetMetricWithLabelValues(methodType, serviceName, methodName)
	}
	for _, code := range allCodes {
		metrics.serverHandledCounter.GetMetricWithLabelValues(methodType, serviceName, methodName, code.String())
	}
}

type grpcType string

const (
	Unary        grpcType = "unary"
	ClientStream grpcType = "client_stream"
	ServerStream grpcType = "server_stream"
	BidiStream   grpcType = "bidi_stream"
)

var allCodes = []codes.Code{
	codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.NotFound,
	codes.AlreadyExists, codes.PermissionDenied, codes.Unauthenticated, codes.ResourceExhausted,
	codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal,
	codes.Unavailable, codes.DataLoss,
}

func splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return "unknown", "unknown"
}

func typeFromMethodInfo(mInfo *grpc.MethodInfo) grpcType {
	if !mInfo.IsClientStream && !mInfo.IsServerStream {
		return Unary
	}
	if mInfo.IsClientStream && !mInfo.IsServerStream {
		return ClientStream
	}
	if !mInfo.IsClientStream && mInfo.IsServerStream {
		return ServerStream
	}
	return BidiStream
}

type serverReporter struct {
	metrics     *ServerMetrics
	rpcType     grpcType
	serviceName string
	methodName  string
	startTime   time.Time
}

func newServerReporter(m *ServerMetrics, rpcType grpcType, fullMethod string) *serverReporter {
	r := &serverReporter{
		metrics: m,
		rpcType: rpcType,
	}
	if r.metrics.serverHandledHistogramEnabled {
		r.startTime = time.Now()
	}
	r.serviceName, r.methodName = splitMethodName(fullMethod)
	r.metrics.serverStartedCounter.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
	return r
}

func (r *serverReporter) ReceivedMessage(ctx context.Context) {
	r.metrics.serverStreamMsgReceivedCounter.WithLabelValues(
		append(
			r.metrics.extension.ServerStreamMsgReceivedCounterValues(ctx),
			string(r.rpcType), r.serviceName, r.methodName,
		)...,
	).Inc()
}

func (r *serverReporter) SentMessage(ctx context.Context) {
	r.metrics.serverStreamMsgSentCounter.WithLabelValues(
		append(
			r.metrics.extension.ServerStreamMsgSentCounterValues(ctx),
			string(r.rpcType), r.serviceName, r.methodName,
		)...,
	).Inc()
}

func (r *serverReporter) Handled(ctx context.Context, code codes.Code) {
	r.metrics.serverHandledCounter.WithLabelValues(
		append(
			r.metrics.extension.ServerHandledCounterValues(ctx),
			string(r.rpcType), r.serviceName, r.methodName, code.String(),
		)...,
	).Inc()

	if r.metrics.serverHandledHistogramEnabled {
		r.metrics.serverHandledHistogram.WithLabelValues(
			append(
				r.metrics.extension.ServerHandledHistogramValues(ctx),
				string(r.rpcType), r.serviceName, r.methodName,
			)...,
		).Observe(time.Since(r.startTime).Seconds())
	}
}
