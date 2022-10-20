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

package backup

import (
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
)

type compatibility struct {
	dbMinVersion         *version.Version
	dbMaxVersion         *version.Version
	backupToolMinVersion *version.Version
	backupToolMaxVersion *version.Version
}

var (
	// ErrIncompatibleService is returned when the service is incompatible for making a backup or restore.
	ErrIncompatibleService = errors.New("incompatible service")
	// ErrXtrabackupNotInstalled is returned if some xtrabackup component is missing.
	ErrXtrabackupNotInstalled = errors.New("xtrabackup is not installed")
	// ErrInvalidXtrabackup is returned if xtrabackup components have different version.
	ErrInvalidXtrabackup = errors.New("invalid installation of the xtrabackup")
	// ErrIncompatibleXtrabackup is returned if xtrabackup is not compatible with the MySQL.
	ErrIncompatibleXtrabackup = errors.New("incompatible xtrabackup")
	// ErrIncompatibleTargetMySQL is returned if target version of MySQL is not compatible for restoring selected artifact.
	ErrIncompatibleTargetMySQL = errors.New("incompatible version of target mysql")
	// ErrComparisonImpossible is returned when comparison of versions is impossible for some reasons.
	ErrComparisonImpossible = errors.New("cannot compare software versions")
	// ErrIncompatibleDataModel is returned if the specified data model (logical or physical) is not compatible with other parameters
	ErrIncompatibleDataModel = errors.New("the specified backup model is not compatible with other parameters")

	mysqlAndXtrabackupCompatibleVersions []compatibility
	// Starting from MySQL 8.0.22 if the Percona XtraBackup version is lower than the database version,
	// processing will be stopped and Percona XtraBackup will not be allowed to continue.
	// https://www.percona.com/blog/2020/08/18/aligning-percona-xtrabackup-versions-with-percona-server-for-mysql/
	alignedVersion = version.Must(version.NewVersion("8.0.22"))
	// Since there is no version 9 or greater let's limit aligning rule by this number.
	maxAlignedVersion = version.Must(version.NewVersion("9.0"))
)

func init() {
	versionStrings := []struct {
		mysqlMinVersion      string // inclusively
		mysqlMaxVersion      string // exclusively
		xtrabackupMinVersion string // inclusively
		xtrabackupMaxVersion string // exclusively
	}{
		// It can back up data from InnoDB, XtraDB, and MyISAM tables on MySQL 5.5, 5.6 and 5.7 servers,
		// as well as Percona Server for MySQL with XtraDB.
		// https://www.percona.com/doc/percona-xtrabackup/2.4/index.html
		{
			mysqlMinVersion:      "5.5",
			mysqlMaxVersion:      "5.8",
			xtrabackupMinVersion: "2.4.18", // https://jira.percona.com/browse/PXB-1978
			xtrabackupMaxVersion: "2.5",
		},
		// In version 8.0.6, Percona XtraBackup introduces the support of the MyRocks storage engine
		// with Percona Server for MySQL version 8.0.15-6 or higher.
		// https://www.percona.com/doc/percona-xtrabackup/8.0/release-notes/8.0/8.0.6.html
		{
			mysqlMinVersion:      "8.0",
			mysqlMaxVersion:      "8.0.20",
			xtrabackupMinVersion: "8.0.6",
			xtrabackupMaxVersion: "9.0",
		},
		// Percona XtraBackup 8.0.12 now supports backup and restore processing for all versions of MySQL;
		// previous versions of Percona XtraBackup will not work with MySQL 8.0.20 and higher.
		// https://www.percona.com/doc/percona-xtrabackup/8.0/release-notes/8.0/8.0.12.html
		// Percona XtraBackup 8.0.13 supports backup and restore processing for all versions of MySQL
		// and has been tested with the latest MySQL 8.0.20.
		// https://www.percona.com/doc/percona-xtrabackup/8.0/release-notes/8.0/8.0.13.html
		{
			mysqlMinVersion:      "8.0.20",
			mysqlMaxVersion:      "8.0.21",
			xtrabackupMinVersion: "8.0.12",
			xtrabackupMaxVersion: "9.0",
		},
		// Percona XtraBackup 8.0.14 supports backup and restore processing for all versions of MySQL
		// and has been tested with the latest MySQL 8.0.21.
		// https://www.percona.com/doc/percona-xtrabackup/8.0/release-notes/8.0/8.0.14.html
		{
			mysqlMinVersion:      "8.0.21",
			mysqlMaxVersion:      "8.0.22",
			xtrabackupMinVersion: "8.0.14",
			xtrabackupMaxVersion: "9.0",
		},
	}

	mysqlAndXtrabackupCompatibleVersions = make([]compatibility, 0, len(versionStrings))
	for _, s := range versionStrings {
		mysqlMinVersion := version.Must(version.NewVersion(s.mysqlMinVersion))
		mysqlMaxVersion := version.Must(version.NewVersion(s.mysqlMaxVersion))
		xtrabackupMinVersion := version.Must(version.NewVersion(s.xtrabackupMinVersion))
		xtrabackupMaxVersion := version.Must(version.NewVersion(s.xtrabackupMaxVersion))

		mysqlAndXtrabackupCompatibleVersions = append(mysqlAndXtrabackupCompatibleVersions, compatibility{
			dbMinVersion:         mysqlMinVersion,
			dbMaxVersion:         mysqlMaxVersion,
			backupToolMinVersion: xtrabackupMinVersion,
			backupToolMaxVersion: xtrabackupMaxVersion,
		})
	}
}

func mysqlAndXtrabackupCompatible(mysqlVersionString, xtrabackupVersionString string) (bool, error) {
	mysqlVersion, err := version.NewVersion(mysqlVersionString)
	if err != nil {
		return false, err
	}
	mysqlVersion = mysqlVersion.Core()

	xtrabackupVersion, err := version.NewVersion(xtrabackupVersionString)
	if err != nil {
		return false, err
	}
	xtrabackupVersion = xtrabackupVersion.Core()

	// See comment to alignedVersion.
	// Using compatibility rule.
	if mysqlVersion.GreaterThanOrEqual(alignedVersion) {
		if xtrabackupVersion.GreaterThanOrEqual(mysqlVersion) && xtrabackupVersion.LessThan(maxAlignedVersion) {
			return true, nil
		}
	} else { // Using compatibility matrix.
		for _, cv := range mysqlAndXtrabackupCompatibleVersions {
			if (mysqlVersion.GreaterThanOrEqual(cv.dbMinVersion) &&
				mysqlVersion.LessThan(cv.dbMaxVersion)) &&
				xtrabackupVersion.GreaterThanOrEqual(cv.backupToolMinVersion) &&
				xtrabackupVersion.LessThan(cv.backupToolMaxVersion) {
				return true, nil
			}
		}
	}
	return false, nil
}

func vendorToServiceType(vendor string) (models.ServiceType, error) {
	serviceType := models.ServiceType(vendor)
	switch serviceType {
	case models.MySQLServiceType,
		models.MongoDBServiceType:
	case models.PostgreSQLServiceType,
		models.ProxySQLServiceType,
		models.HAProxyServiceType,
		models.ExternalServiceType:
		return "", status.Errorf(codes.Unimplemented, "unimplemented service type: %s", serviceType)
	default:
		return "", status.Errorf(codes.Internal, "unknown service type: %s", serviceType)
	}

	return serviceType, nil
}

func softwareVersionsToMap(svs models.SoftwareVersions) map[models.SoftwareName]string {
	m := make(map[models.SoftwareName]string, len(svs))
	for _, sv := range svs {
		m[sv.Name] = sv.Version
	}
	return m
}

func mySQLSoftwaresInstalledAndCompatible(svm map[models.SoftwareName]string) error {
	for _, name := range []models.SoftwareName{
		models.MysqldSoftwareName,
		models.XtrabackupSoftwareName,
		models.XbcloudSoftwareName,
		models.QpressSoftwareName,
	} {
		if svm[name] == "" {
			if name == models.XtrabackupSoftwareName || name == models.XbcloudSoftwareName {
				return errors.Wrapf(ErrXtrabackupNotInstalled, "software %q is not installed", name)
			}

			return errors.Wrapf(ErrIncompatibleService, "software %q is not installed", name)
		}
	}

	if svm[models.XtrabackupSoftwareName] != svm[models.XbcloudSoftwareName] {
		return errors.Wrapf(ErrInvalidXtrabackup, "xtrabackup version %q != xbcloud version %q",
			svm[models.XtrabackupSoftwareName], svm[models.XbcloudSoftwareName])
	}

	ok, err := mysqlAndXtrabackupCompatible(svm[models.MysqldSoftwareName], svm[models.XtrabackupSoftwareName])
	if err != nil {
		return err
	}
	if !ok {
		return errors.Wrapf(ErrIncompatibleXtrabackup, "xtrabackup version %q is not compatible with mysql version %q",
			svm[models.XtrabackupSoftwareName], svm[models.MysqldSoftwareName])
	}

	return nil
}

func convertSoftwareName(s agents.Software) (models.SoftwareName, error) {
	var softwareName models.SoftwareName
	switch software := s.(type) {
	case *agents.Mysqld:
		softwareName = models.MysqldSoftwareName
	case *agents.Xtrabackup:
		softwareName = models.XtrabackupSoftwareName
	case *agents.Xbcloud:
		softwareName = models.XbcloudSoftwareName
	case *agents.Qpress:
		softwareName = models.QpressSoftwareName
	default:
		return "", errors.Errorf("invalid software type %T", software)
	}

	return softwareName, nil
}

// isOnlySameService checks if restore is only available to the same service.
func isOnlySameService(artifactDBVersion string, serviceType models.ServiceType) bool {
	// allow restore to the same service if db version is unknown or service type is MongoDB.
	if artifactDBVersion == "" || serviceType == models.MongoDBServiceType {
		return true
	}
	return false
}
