// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package supervisor

import "regexp"

var (
	nodeExporterRegexp         = regexp.MustCompile("node_exporter, version ([!-~]*).*")
	mysqldExporterRegexp       = regexp.MustCompile("mysqld_exporter, version ([!-~]*).*")
	postgresExporterRegexp     = regexp.MustCompile("postgres_exporter, version ([!-~]*).*")
	proxysqlExporterRegexp     = regexp.MustCompile("proxysql_exporter, version ([!-~]*).*")
	rdsExporterRegexp          = regexp.MustCompile("rds_exporter, version ([!-~]*).*")
	azureMetricsExporterRegexp = regexp.MustCompile("azure_metrics_exporter, version ([!-~]*).*")
	valkeyExporterRegexp       = regexp.MustCompile("valkey_exporter, version ([!-~]*).*")
	mongodbExporterRegexp      = regexp.MustCompile("Version: ([!-~]*).*")
)

// agentVersioner is a subset of methods of version.Versioner used by this package.
type agentVersioner interface {
	BinaryVersion(
		binaryName string,
		expectedExitCode int,
		versionRegexp *regexp.Regexp,
		arg ...string,
	) (string, error)
}
