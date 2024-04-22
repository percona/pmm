// Copyright (C) 2024 Percona LLC
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

type (
	// Cluster contains information about how to communicate with a kubernetes cluster.
	Cluster struct {
		CertificateAuthorityData []byte `json:"certificate-authority-data"`
		Server                   string `json:"server"`
	}
	// ClusterInfo is a struct used to parse Cluster config from kubeconfig.
	ClusterInfo struct {
		Name    string  `json:"name"`
		Cluster Cluster `json:"cluster"`
	}
	// User contains information that describes identity information.  This is use to tell the kubernetes cluster who you are.
	User struct {
		Token string `json:"token"`
	}
	// UserInfo is a struct used to parse User config from kubeconfig.
	UserInfo struct {
		Name string `json:"name"`
		User User   `json:"user"`
	}
	// Context is a tuple of references to a cluster (how do I communicate with a kubernetes cluster),
	// a user (how do I identify myself), and a namespace (what subset of resources do I want to work with).
	Context struct {
		Cluster   string `json:"cluster"`
		User      string `json:"user"`
		Namespace string `json:"namespace"`
	}
	// ContextInfo is a struct used to parse Context config from kubeconfig.
	ContextInfo struct {
		Name    string  `json:"name"`
		Context Context `json:"context"`
	}
	// Config holds the information needed to build connect to remote kubernetes clusters as a given user.
	Config struct {
		// Legacy field from pkg/api/types.go TypeMeta.
		Kind string `json:"kind,omitempty"`
		// Legacy field from pkg/api/types.go TypeMeta.
		APIVersion string `json:"apiVersion,omitempty"`
		// Preferences holds general information to be use for cli interactions
		Clusters []ClusterInfo `json:"clusters"`
		// AuthInfos is a map of referencable names to user configs
		Users []UserInfo `json:"users"`
		// Contexts is a map of referencable names to context configs
		Contexts []ContextInfo `json:"contexts"`
		// CurrentContext is the name of the context that you would like to use by default
		CurrentContext string `json:"current-context"`
	}
)
