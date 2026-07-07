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

package backup

import (
	"fmt"

	"github.com/go-faster/errors"
	"github.com/hashicorp/go-version"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/models"
)

type compatibility struct {
	dbVersions         versionRange
	backupToolVersions versionRange
}

type versionRange struct {
	min *version.Version
	max *version.Version
}

func (r versionRange) isSupported(v *version.Version) bool {
	return !v.LessThan(r.min) && v.LessThan(r.max)
}

func mustVersion(versionString string) *version.Version {
	return version.Must(version.NewVersion(versionString))
}

var (
	// Starting from MySQL 8.0.22 if the Percona XtraBackup version is lower than the database version,
	// processing will be stopped and Percona XtraBackup will not be allowed to continue.
	// https://www.percona.com/blog/2020/08/18/aligning-percona-xtrabackup-versions-with-percona-server-for-mysql/
	alignedXtrabackupVersion = mustVersion("8.0.22")
	// Starting with Percona XtraBackup 8.0.34, any 8.0.34+ PXB can back up 8.0.34+
	// servers within the 8.0 series.
	// https://docs.percona.com/percona-xtrabackup/8.0/release-notes/8.0/8.0.34-29.0.html
	universal80Version = mustVersion("8.0.34")

	mysql81Version = mustVersion("8.1.0")
	mysql82Version = mustVersion("8.2.0")
	mysql83Version = mustVersion("8.3.0")
	// Any Percona XtraBackup 8.4 release can work with any MySQL 8.4 release.
	// https://docs.percona.com/percona-xtrabackup/8.4/xtrabackup-version-numbers.html
	mysql84Version = mustVersion("8.4.0")
	mysql85Version = mustVersion("8.5.0")

	pbmMinSupportedVersion = mustVersion("2.0.1")

	mysqlAndXtrabackupCompatibleVersions = []compatibility{
		// It can back up data from InnoDB, XtraDB, and MyISAM tables on MySQL 5.5, 5.6 and 5.7 servers,
		// as well as Percona Server for MySQL with XtraDB.
		// https://www.percona.com/doc/percona-xtrabackup/2.4/index.html
		{
			dbVersions: versionRange{min: mustVersion("5.5"), max: mustVersion("5.8")},
			// https://jira.percona.com/browse/PXB-1978
			backupToolVersions: versionRange{min: mustVersion("2.4.18"), max: mustVersion("2.5")},
		},
		// In version 8.0.6, Percona XtraBackup introduces the support of the MyRocks storage engine
		// with Percona Server for MySQL version 8.0.15-6 or higher.
		// https://www.percona.com/doc/percona-xtrabackup/8.0/release-notes/8.0/8.0.6.html
		{
			dbVersions:         versionRange{min: mustVersion("8.0"), max: mustVersion("8.0.20")},
			backupToolVersions: versionRange{min: mustVersion("8.0.6"), max: mustVersion("8.1.0")},
		},
		// Percona XtraBackup 8.0.12 now supports backup and restore processing for all versions of MySQL;
		// previous versions of Percona XtraBackup will not work with MySQL 8.0.20 and higher.
		// https://www.percona.com/doc/percona-xtrabackup/8.0/release-notes/8.0/8.0.12.html
		// Percona XtraBackup 8.0.13 supports backup and restore processing for all versions of MySQL
		// and has been tested with the latest MySQL 8.0.20.
		// https://www.percona.com/doc/percona-xtrabackup/8.0/release-notes/8.0/8.0.13.html
		{
			dbVersions:         versionRange{min: mustVersion("8.0.20"), max: mustVersion("8.0.21")},
			backupToolVersions: versionRange{min: mustVersion("8.0.12"), max: mustVersion("8.1.0")},
		},
		// Percona XtraBackup 8.0.14 supports backup and restore processing for all versions of MySQL
		// and has been tested with the latest MySQL 8.0.21.
		// https://www.percona.com/doc/percona-xtrabackup/8.0/release-notes/8.0/8.0.14.html
		{
			dbVersions:         versionRange{min: mustVersion("8.0.21"), max: mustVersion("8.0.22")},
			backupToolVersions: versionRange{min: mustVersion("8.0.14"), max: mustVersion("8.1.0")},
		},
	}
)

// mysqlAndXtrabackupCompatible is kept for compatibility matrix tests; production code uses
// mysqlAndXtrabackupCompatibilityError to return user-facing guidance.
func mysqlAndXtrabackupCompatible(mysqlVersionString, xtrabackupVersionString string) (bool, error) {
	mysqlVersion, xtrabackupVersion, err := mysqlAndXtrabackupCoreVersions(mysqlVersionString, xtrabackupVersionString)
	if err != nil {
		return false, err
	}

	return mysqlAndXtrabackupCoreVersionsCompatible(mysqlVersion, xtrabackupVersion), nil
}

func mysqlAndXtrabackupCoreVersions(mysqlVersionString, xtrabackupVersionString string) (*version.Version, *version.Version, error) {
	mysqlVersion, err := version.NewVersion(mysqlVersionString)
	if err != nil {
		return nil, nil, err
	}
	mysqlVersion = mysqlVersion.Core()

	xtrabackupVersion, err := version.NewVersion(xtrabackupVersionString)
	if err != nil {
		return nil, nil, err
	}
	xtrabackupVersion = xtrabackupVersion.Core()

	return mysqlVersion, xtrabackupVersion, nil
}

type mysqlXtrabackupBand int

const (
	mysqlXtrabackupBand84 mysqlXtrabackupBand = iota
	mysqlXtrabackupBand83
	mysqlXtrabackupBand82
	mysqlXtrabackupBand81
	mysqlXtrabackupBand80Universal
	mysqlXtrabackupBand80Aligned
	mysqlXtrabackupBandLegacy
	mysqlXtrabackupBandUnsupported
)

func mysqlXtrabackupBandFor(mysqlVersion *version.Version) mysqlXtrabackupBand {
	switch {
	case versionRange{min: mysql84Version, max: mysql85Version}.isSupported(mysqlVersion):
		return mysqlXtrabackupBand84
	case versionRange{min: mysql83Version, max: mysql84Version}.isSupported(mysqlVersion):
		return mysqlXtrabackupBand83
	case versionRange{min: mysql82Version, max: mysql83Version}.isSupported(mysqlVersion):
		return mysqlXtrabackupBand82
	case versionRange{min: mysql81Version, max: mysql82Version}.isSupported(mysqlVersion):
		return mysqlXtrabackupBand81
	case versionRange{min: universal80Version, max: mysql81Version}.isSupported(mysqlVersion):
		return mysqlXtrabackupBand80Universal
	case versionRange{min: alignedXtrabackupVersion, max: universal80Version}.isSupported(mysqlVersion):
		return mysqlXtrabackupBand80Aligned
	case mysqlVersion.LessThan(alignedXtrabackupVersion):
		return mysqlXtrabackupBandLegacy
	default:
		return mysqlXtrabackupBandUnsupported
	}
}

func incompatibleXtrabackupError(message, xtrabackupVersionString, mysqlVersionString string) error {
	return errors.Wrapf(
		ErrIncompatibleXtrabackup,
		message,
		xtrabackupVersionString,
		mysqlVersionString,
	)
}

func mysqlAndXtrabackupCoreVersionsCompatible(mysqlVersion, xtrabackupVersion *version.Version) bool {
	return mysqlAndXtrabackupCoreVersionsCompatibleForBand(
		mysqlXtrabackupBandFor(mysqlVersion),
		mysqlVersion,
		xtrabackupVersion,
	)
}

func mysqlAndXtrabackupCoreVersionsCompatibleForBand(
	band mysqlXtrabackupBand,
	mysqlVersion, xtrabackupVersion *version.Version,
) bool {
	switch band {
	case mysqlXtrabackupBand84:
		return versionRange{min: mysql84Version, max: mysql85Version}.isSupported(xtrabackupVersion)
	case mysqlXtrabackupBand83:
		return versionRange{min: mysql83Version, max: mysql84Version}.isSupported(xtrabackupVersion)
	case mysqlXtrabackupBand82:
		return versionRange{min: mysql82Version, max: mysql83Version}.isSupported(xtrabackupVersion)
	case mysqlXtrabackupBand81:
		return versionRange{min: mysql81Version, max: mysql82Version}.isSupported(xtrabackupVersion)
	case mysqlXtrabackupBand80Universal:
		return versionRange{min: universal80Version, max: mysql81Version}.isSupported(xtrabackupVersion)
	case mysqlXtrabackupBand80Aligned:
		return versionRange{min: mysqlVersion, max: mysql81Version}.isSupported(xtrabackupVersion)
	case mysqlXtrabackupBandLegacy:
		for _, cv := range mysqlAndXtrabackupCompatibleVersions {
			if cv.dbVersions.isSupported(mysqlVersion) && cv.backupToolVersions.isSupported(xtrabackupVersion) {
				return true
			}
		}
	case mysqlXtrabackupBandUnsupported:
		return false
	}

	return false
}

func mysqlAndXtrabackupCompatibilityError(mysqlVersionString, xtrabackupVersionString string) error {
	mysqlVersion, xtrabackupVersion, err := mysqlAndXtrabackupCoreVersions(mysqlVersionString, xtrabackupVersionString)
	if err != nil {
		return err
	}
	band := mysqlXtrabackupBandFor(mysqlVersion)
	if mysqlAndXtrabackupCoreVersionsCompatibleForBand(band, mysqlVersion, xtrabackupVersion) {
		return nil
	}

	switch band {
	case mysqlXtrabackupBand84:
		return incompatibleXtrabackupError(
			"Percona XtraBackup version %q is not compatible with MySQL version %q; use Percona XtraBackup 8.4.x for MySQL 8.4.x",
			xtrabackupVersionString,
			mysqlVersionString,
		)
	case mysqlXtrabackupBand83:
		return incompatibleXtrabackupError(
			"Percona XtraBackup version %q is not compatible with MySQL version %q; use Percona XtraBackup 8.3.x for MySQL 8.3.x",
			xtrabackupVersionString,
			mysqlVersionString,
		)
	case mysqlXtrabackupBand82:
		return incompatibleXtrabackupError(
			"Percona XtraBackup version %q is not compatible with MySQL version %q; use Percona XtraBackup 8.2.x for MySQL 8.2.x",
			xtrabackupVersionString,
			mysqlVersionString,
		)
	case mysqlXtrabackupBand81:
		return incompatibleXtrabackupError(
			"Percona XtraBackup version %q is not compatible with MySQL version %q; use Percona XtraBackup 8.1.x for MySQL 8.1.x",
			xtrabackupVersionString,
			mysqlVersionString,
		)
	case mysqlXtrabackupBand80Universal:
		return incompatibleXtrabackupError(
			"Percona XtraBackup version %q is not compatible with MySQL version %q; use Percona XtraBackup 8.0.34 or newer 8.0.x for MySQL 8.0.34+",
			xtrabackupVersionString,
			mysqlVersionString,
		)
	case mysqlXtrabackupBand80Aligned:
		if xtrabackupVersion.LessThan(mysqlVersion) {
			return incompatibleXtrabackupError(
				"Percona XtraBackup version %q is older than MySQL version %q; "+
					"for MySQL 8.0.22+, use Percona XtraBackup 8.0.x with the same or newer core version",
				xtrabackupVersionString,
				mysqlVersionString,
			)
		}

		return incompatibleXtrabackupError(
			"Percona XtraBackup version %q is not compatible with MySQL version %q; use Percona XtraBackup 8.0.x for MySQL 8.0.x",
			xtrabackupVersionString,
			mysqlVersionString,
		)
	case mysqlXtrabackupBandLegacy:
		return incompatibleXtrabackupError(
			"Percona XtraBackup version %q is not compatible with MySQL version %q; "+
				"install a Percona XtraBackup version supported for this MySQL version",
			xtrabackupVersionString,
			mysqlVersionString,
		)
	default:
		return incompatibleXtrabackupError(
			"PMM does not support Percona XtraBackup version %q with MySQL version %q yet",
			xtrabackupVersionString,
			mysqlVersionString,
		)
	}
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

func mySQLBackupSoftwareInstalledAndCompatible(svm map[models.SoftwareName]string) error {
	for _, name := range []models.SoftwareName{
		models.MysqldSoftwareName,
		models.XtrabackupSoftwareName,
		models.XbcloudSoftwareName,
		models.QpressSoftwareName,
	} {
		if svm[name] == "" {
			if name == models.XtrabackupSoftwareName || name == models.XbcloudSoftwareName {
				return fmt.Errorf("software %q is not installed: %w", name, ErrXtrabackupNotInstalled)
			}

			return fmt.Errorf("software %q is not installed: %w", name, ErrIncompatibleService)
		}
	}

	if svm[models.XtrabackupSoftwareName] != svm[models.XbcloudSoftwareName] {
		return fmt.Errorf("xtrabackup version %q != xbcloud version %q: %w",
			svm[models.XtrabackupSoftwareName], svm[models.XbcloudSoftwareName], ErrInvalidXtrabackup)
	}

	err := mysqlAndXtrabackupCompatibilityError(svm[models.MysqldSoftwareName], svm[models.XtrabackupSoftwareName])
	if err != nil {
		return err
	}

	return nil
}

func mongoDBBackupSoftwareInstalledAndCompatible(svm map[models.SoftwareName]string) error {
	for _, name := range []models.SoftwareName{
		models.MongoDBSoftwareName,
		models.PBMSoftwareName,
	} {
		if svm[name] == "" {
			return fmt.Errorf("software %q is not installed: %w", name, ErrIncompatibleService)
		}
	}

	pbmVersion, err := version.NewVersion(svm[models.PBMSoftwareName])
	if err != nil {
		return err
	}
	pbmVersion = pbmVersion.Core()

	if pbmVersion.LessThan(pbmMinSupportedVersion) {
		return fmt.Errorf("installed pbm version %q, min required pbm version %q: %w", pbmVersion, pbmMinSupportedVersion, ErrIncompatiblePBM)
	}

	return nil
}

// isOnlySameService checks if restore is only available to the same service.
func isOnlySameService(artifactDBVersion string) bool {
	// allow restore only to the same service if db version is unknown.
	return artifactDBVersion == ""
}
