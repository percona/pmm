// Copyright 2019 Percona LLC
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

package gke

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
)

// InstallCommand is used by Kong for CLI flags and commands.
type InstallCommand struct {
	Name string `default:"michal-dbaas"`
}

type installResult struct{}

// Result is a command run result.
func (res *installResult) Result() {}

// String stringifies command result.
func (res *installResult) String() string {
	return "works"
}

const location = "europe-west1-b"

// RunCmdWithContext runs install command.
func (c *InstallCommand) RunCmdWithContext(ctx context.Context, flags *flags.GlobalFlags) (commands.Result, error) {
	start := time.Now()

	logrus.Info("Creating GKE")

	cl, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	containerService, err := container.New(cl)
	if err != nil {
		return nil, err
	}

	op, err := c.createGKECluster(ctx, containerService)
	if err != nil {
		return nil, err
	}

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		t := time.NewTicker(5 * time.Second)
		name := "projects/percona-gcp-dev/locations/" + location + "/operations/"

		for {
			<-t.C
			op, err := containerService.Projects.Locations.Operations.Get(name + op.Name).Context(ctx).Do()
			if err != nil {
				logrus.Info(err)
			}

			if op.Status == "DONE" {
				return
			}

			for _, m := range op.Progress.Metrics {
				logrus.Infof("%#v", m)
			}
		}
	}()

	<-ch

	logrus.Infof("Elapsed time %s\n", time.Since(start))
	logrus.Info("Getting credentials")
	cmd := exec.Command(
		"gcloud",
		"container",
		"clusters",
		"get-credentials",
		c.Name,
		"--zone="+location,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	logrus.Infof("Elapsed time %s\n", time.Since(start))
	logrus.Info("Running kubectl")
	cmd = exec.Command("kubectl", "apply", "-f", "/home/michal/pmm-server.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	logrus.Infof("Elapsed time %s\n", time.Since(start))

	ipChan := c.getIngressIp()
	ip := <-ipChan

	logrus.Infof("Elapsed time %s\n", time.Since(start))
	logrus.Infof("Got IP %s", ip)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range time.Tick(5 * time.Second) {
			logrus.Info("Checking ingress health")
			res, err := http.Get("http://" + ip)
			res.Body.Close()

			if err != nil {
				logrus.Error(err)
				continue
			}

			if res.StatusCode >= 500 {
				continue
			}

			return
		}
	}()

	<-done

	logrus.Infof("Elapsed time %s\n", time.Since(start))
	logrus.Info("Visit http://" + ip)

	return &installResult{}, nil
}

func (c *InstallCommand) createGKECluster(ctx context.Context, containerService *container.Service) (*container.Operation, error) {
	parent := "projects/percona-gcp-dev/locations/" + location

	rb := &container.CreateClusterRequest{
		Cluster: &container.Cluster{
			Name:             c.Name,
			InitialNodeCount: 3,
			NodeConfig: &container.NodeConfig{
				Preemptible: true,
				MachineType: "e2-standard-4",
			},
		},
	}

	resp, err := containerService.Projects.Locations.Clusters.Create(parent, rb).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type getIngress struct {
	Status *struct {
		LoadBalancer *struct {
			Ingress []struct {
				Ip string `json:"ip"`
			} `json:"ingress"`
		} `json:"loadBalancer"`
	} `json:"status"`
}

func (c *InstallCommand) getIngressIp() <-chan string {
	done := make(chan string)
	go func() {
		defer close(done)

		t := time.NewTicker(5 * time.Second)
		for {
			<-t.C
			logrus.Info("Checking IP")
			cmd := exec.Command("kubectl", "get", "ing", "pmm-http", "-o", "json")
			var b bytes.Buffer
			cmd.Stdout = &b

			if err := cmd.Run(); err != nil {
				continue
			}

			res := getIngress{}
			if err := json.Unmarshal(b.Bytes(), &res); err != nil {
				logrus.Error(err)
				continue
			}

			if res.Status != nil &&
				res.Status.LoadBalancer != nil &&
				res.Status.LoadBalancer.Ingress != nil &&
				len(res.Status.LoadBalancer.Ingress) > 0 {
				done <- res.Status.LoadBalancer.Ingress[0].Ip
				return
			}
		}
	}()

	return done
}
