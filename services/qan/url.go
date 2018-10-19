// pmm-managed
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

package qan

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/utils/logger"
)

/*
We almost could connect to 127.0.0.1:9001 and do not use nginx and HTTP Basic auth.
Unfortunately, QAN API always assumes /qan-api/ in URLs in responses, even if pmm-managed goes directly to it.
See https://github.com/percona/qan-api/blob/v1.4.1/app/init.go#L156-L166
In the end, qan-agent receives something like ws://127.0.0.1:9001/qan-api which doesn't work.

So instead we get nginx username and password from file (if it is present) and pass those credentials everywhere.
*/

type manageConfig struct {
	Users []struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"users"`
}

func getQanURL(ctx context.Context) (*url.URL, error) {
	pmmQanApiURL := os.Getenv("PMM_QAN_API_URL")
	if pmmQanApiURL != "" {
		return url.Parse(pmmQanApiURL)
	}

	u := &url.URL{
		Scheme: "http",
		Host:   "127.0.0.1",
		Path:   "/qan-api/",
	}
	f, err := os.Open("/srv/update/pmm-manage.yml")
	if err != nil {
		if os.IsNotExist(err) {
			logger.Get(ctx).WithField("component", "qan").Info("pmm-manage.yml not found, assuming default QAN API URL.")
			return u, nil
		}
		return nil, errors.WithStack(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var config manageConfig
	if err = yaml.Unmarshal(b, &config); err != nil {
		return nil, errors.WithStack(err)
	}
	if len(config.Users) > 0 && config.Users[0].Username != "" {
		u.User = url.UserPassword(config.Users[0].Username, config.Users[0].Password)
	}
	return u, nil
}
