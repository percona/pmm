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

// Package grpc provides gRPC server implementation for real-time analytics.
package grpc

import (
	"context"

	realtimev1 "github.com/percona/pmm/api/realtime/v1"
)

// realtimeService is a subset of the real-time analytics service.
type realtimeService interface {
	SendRealTimeData(ctx context.Context, req *realtimev1.RealTimeAnalyticsRequest) (*realtimev1.RealTimeAnalyticsResponse, error)
	GetRealTimeData(ctx context.Context, req *realtimev1.RealTimeDataRequest) (*realtimev1.RealTimeDataResponse, error)
	EnableRealTimeAnalytics(ctx context.Context, req *realtimev1.EnableRealTimeAnalyticsRequest) (*realtimev1.ConfigResponse, error)
	DisableRealTimeAnalytics(ctx context.Context, req *realtimev1.DisableRealTimeAnalyticsRequest) (*realtimev1.ConfigResponse, error)
}

// RealTimeServer implements the real-time analytics gRPC server.
type RealTimeServer struct {
	realtimev1.UnimplementedRealTimeAnalyticsServiceServer

	s realtimeService
}

// NewRealTimeServer creates a new real-time analytics gRPC server.
func NewRealTimeServer(s realtimeService) *RealTimeServer {
	return &RealTimeServer{
		s: s,
	}
}

// SendRealTimeData handles incoming real-time data from agents.
func (rs *RealTimeServer) SendRealTimeData(ctx context.Context, req *realtimev1.RealTimeAnalyticsRequest) (*realtimev1.RealTimeAnalyticsResponse, error) {
	return rs.s.SendRealTimeData(ctx, req)
}

// GetRealTimeData retrieves current real-time data for the UI.
func (rs *RealTimeServer) GetRealTimeData(ctx context.Context, req *realtimev1.RealTimeDataRequest) (*realtimev1.RealTimeDataResponse, error) {
	return rs.s.GetRealTimeData(ctx, req)
}

// EnableRealTimeAnalytics enables real-time analytics for a service.
func (rs *RealTimeServer) EnableRealTimeAnalytics(ctx context.Context, req *realtimev1.EnableRealTimeAnalyticsRequest) (*realtimev1.ConfigResponse, error) {
	return rs.s.EnableRealTimeAnalytics(ctx, req)
}

// DisableRealTimeAnalytics disables real-time analytics for a service.
func (rs *RealTimeServer) DisableRealTimeAnalytics(ctx context.Context, req *realtimev1.DisableRealTimeAnalyticsRequest) (*realtimev1.ConfigResponse, error) {
	return rs.s.DisableRealTimeAnalytics(ctx, req)
}
