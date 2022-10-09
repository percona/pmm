// Copyright (C) 2017 Percona LLC
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

package main

import (
	"fmt"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	alertingpb "github.com/percona/pmm/api/managementpb/alerting"
	azurev1beta1 "github.com/percona/pmm/api/managementpb/azure"
	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/percona/pmm/api/platformpb"
	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/api/userpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func main() {
	telemetryGenerateApiUsage()
}

func telemetryGenerateApiUsage() {
	gRPCServer := grpc.NewServer()

	registerUnimplementedServer(gRPCServer)

	getMetrics(gRPCServer)
}

func getMetrics(server *grpc.Server) {
	serviceInfo := server.GetServiceInfo()
	for serviceName, info := range serviceInfo {
		for _, mInfo := range info.Methods {
			methodName := mInfo.Name
			methodType := typeFromMethodInfo(mInfo)

			for _, code := range allCodes {
				fmt.Println(prometheusMetricName(methodType, serviceName, methodName, code.String()))
			}
		}
	}
}

var (
	allCodes = []codes.Code{
		codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.NotFound,
		codes.AlreadyExists, codes.PermissionDenied, codes.Unauthenticated, codes.ResourceExhausted,
		codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal,
		codes.Unavailable, codes.DataLoss,
	}
)

func prometheusMetricName(methodType, serviceName, methodName, code string) string {
	return fmt.Sprintf(`grpc_server_handled_total{grpc_code=%q,grpc_method=%q,grpc_service=%q,grpc_type=%q}`,
		code, methodName, serviceName, methodType,
	)
}

func typeFromMethodInfo(mInfo grpc.MethodInfo) string {
	if !mInfo.IsClientStream && !mInfo.IsServerStream {
		return "unary"
	}
	if mInfo.IsClientStream && !mInfo.IsServerStream {
		return "client_stream"
	}
	if !mInfo.IsClientStream && mInfo.IsServerStream {
		return "server_stream"
	}
	return "bidi_stream"
}

func registerUnimplementedServer(gRPCServer grpc.ServiceRegistrar) {
	serverpb.RegisterServerServer(gRPCServer, new(serverpb.UnimplementedServerServer))

	agentpb.RegisterAgentServer(gRPCServer, new(agentpb.UnimplementedAgentServer))

	inventorypb.RegisterNodesServer(gRPCServer, new(inventorypb.UnimplementedNodesServer))
	inventorypb.RegisterServicesServer(gRPCServer, new(inventorypb.UnimplementedServicesServer))
	inventorypb.RegisterAgentsServer(gRPCServer, new(inventorypb.UnimplementedAgentsServer))

	managementpb.RegisterNodeServer(gRPCServer, new(managementpb.UnimplementedNodeServer))
	managementpb.RegisterServiceServer(gRPCServer, new(managementpb.UnimplementedServiceServer))
	managementpb.RegisterMySQLServer(gRPCServer, new(managementpb.UnimplementedMySQLServer))
	managementpb.RegisterMongoDBServer(gRPCServer, new(managementpb.UnimplementedMongoDBServer))
	managementpb.RegisterPostgreSQLServer(gRPCServer, new(managementpb.UnimplementedPostgreSQLServer))
	managementpb.RegisterProxySQLServer(gRPCServer, new(managementpb.UnimplementedProxySQLServer))
	managementpb.RegisterActionsServer(gRPCServer, new(managementpb.UnimplementedActionsServer))
	managementpb.RegisterRDSServer(gRPCServer, new(managementpb.UnimplementedRDSServer))
	azurev1beta1.RegisterAzureDatabaseServer(gRPCServer, new(azurev1beta1.UnimplementedAzureDatabaseServer))
	managementpb.RegisterHAProxyServer(gRPCServer, new(managementpb.UnimplementedHAProxyServer))
	managementpb.RegisterExternalServer(gRPCServer, new(managementpb.UnimplementedExternalServer))
	managementpb.RegisterAnnotationServer(gRPCServer, new(managementpb.UnimplementedAnnotationServer))
	managementpb.RegisterSecurityChecksServer(gRPCServer, new(managementpb.UnimplementedSecurityChecksServer))

	iav1beta1.RegisterChannelsServer(gRPCServer, new(iav1beta1.UnimplementedChannelsServer))
	iav1beta1.RegisterRulesServer(gRPCServer, new(iav1beta1.UnimplementedRulesServer))
	iav1beta1.RegisterAlertsServer(gRPCServer, new(iav1beta1.UnimplementedAlertsServer))
	alertingpb.RegisterAlertingServer(gRPCServer, new(alertingpb.UnimplementedAlertingServer))

	backupv1beta1.RegisterBackupsServer(gRPCServer, new(backupv1beta1.UnimplementedBackupsServer))
	backupv1beta1.RegisterLocationsServer(gRPCServer, new(backupv1beta1.UnimplementedLocationsServer))
	backupv1beta1.RegisterArtifactsServer(gRPCServer, new(backupv1beta1.UnimplementedArtifactsServer))
	backupv1beta1.RegisterRestoreHistoryServer(gRPCServer, new(backupv1beta1.UnimplementedRestoreHistoryServer))

	dbaasv1beta1.RegisterKubernetesServer(gRPCServer, new(dbaasv1beta1.UnimplementedKubernetesServer))
	dbaasv1beta1.RegisterDBClustersServer(gRPCServer, new(dbaasv1beta1.UnimplementedDBClustersServer))
	dbaasv1beta1.RegisterPXCClustersServer(gRPCServer, new(dbaasv1beta1.UnimplementedPXCClustersServer))
	dbaasv1beta1.RegisterPSMDBClustersServer(gRPCServer, new(dbaasv1beta1.UnimplementedPSMDBClustersServer))
	dbaasv1beta1.RegisterLogsAPIServer(gRPCServer, new(dbaasv1beta1.UnimplementedLogsAPIServer))
	dbaasv1beta1.RegisterComponentsServer(gRPCServer, new(dbaasv1beta1.UnimplementedComponentsServer))

	userpb.RegisterUserServer(gRPCServer, new(userpb.UnimplementedUserServer))
	platformpb.RegisterPlatformServer(gRPCServer, new(platformpb.UnimplementedPlatformServer))
}
