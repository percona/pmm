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

package supervisord

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// vmCredentialsDir holds the username and password that the VictoriaMetrics URL (PMM_VM_URL) carries,
// for vmalert and vmproxy. They read them from files, so that the credentials are neither in their
// configuration nor on their command line. The files are written before supervisord starts them, since
// vmalert does not start without them, and are not kept in /srv: they come from the environment.
const vmCredentialsDir = "/tmp/pmm-vm-credentials" //nolint:gosec

// vmCredentialsDirPermissions lets only PMM Server's user into vmCredentialsDir: a directory needs the
// execute bit to be entered.
const vmCredentialsDirPermissions = 0o700

var (
	vmUsernameFile = filepath.Join(vmCredentialsDir, "username")
	vmPasswordFile = filepath.Join(vmCredentialsDir, "password")
)

// SaveVMCredentials writes the username and password that the VictoriaMetrics URL carries to the files
// that vmalert and vmproxy read them from, and removes the files if it carries none.
func SaveVMCredentials(vmURL *url.URL) error {
	return saveVMCredentials(vmCredentialsDir, vmURL)
}

func saveVMCredentials(dir string, vmURL *url.URL) error {
	if vmURL.User == nil {
		err := os.RemoveAll(dir)
		if err != nil {
			return fmt.Errorf("cannot remove the VictoriaMetrics credentials: %w", err)
		}
		return nil
	}

	err := os.MkdirAll(dir, vmCredentialsDirPermissions)
	if err == nil {
		// It may be left from an earlier start, with other permissions.
		err = os.Chmod(dir, vmCredentialsDirPermissions)
	}
	if err != nil {
		return fmt.Errorf("cannot create the directory of the VictoriaMetrics credentials: %w", err)
	}

	password, _ := vmURL.User.Password()
	var errs []error
	for name, value := range map[string]string{"username": vmURL.User.Username(), "password": password} {
		errs = append(errs, writeSecretFile(filepath.Join(dir, name), value))
	}
	err = errors.Join(errs...)
	if err != nil {
		return fmt.Errorf("cannot write the VictoriaMetrics credentials: %w", err)
	}
	return nil
}

// writeSecretFile replaces the file at path with one that only its owner can read, holding value.
func writeSecretFile(path, value string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck

	_, err = tmp.WriteString(value)
	if err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// vmURLWithoutCredentials returns the VictoriaMetrics URL for vmalert and vmproxy, without the
// credentials it may carry, and the files with the credentials, which are empty if it carries none.
func vmURLWithoutCredentials(vmURL *url.URL) (string, string, string) {
	if vmURL.User == nil {
		return vmURL.String(), "", ""
	}

	stripped := *vmURL
	stripped.User = nil
	return stripped.String(), vmUsernameFile, vmPasswordFile
}
