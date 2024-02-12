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

package client

import (
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// configGetter stores kubeconfig string to convert it to the final object.
type configGetter struct {
	kubeconfig string
}

// NewConfigGetter creates a new configGetter struct.
func NewConfigGetter(kubeconfig string) *configGetter { //nolint:revive
	return &configGetter{kubeconfig: kubeconfig}
}

// loadFromString takes a kubeconfig and deserializes the contents into Config object.
func (g *configGetter) loadFromString() (*clientcmdapi.Config, error) {
	config, err := clientcmd.Load([]byte(g.kubeconfig))
	if err != nil {
		return nil, err
	}

	if config.AuthInfos == nil {
		config.AuthInfos = make(map[string]*clientcmdapi.AuthInfo)
	}
	if config.Clusters == nil {
		config.Clusters = make(map[string]*clientcmdapi.Cluster)
	}
	if config.Contexts == nil {
		config.Contexts = make(map[string]*clientcmdapi.Context)
	}

	return config, nil
}
