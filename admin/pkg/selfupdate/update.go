// Copyright 2023 Percona LLC
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

// Package selfupdate holds logic to self updating pmm-server-upgrade.
package selfupdate

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
)

const (
	dateSuffixFormat = "2006-01-02-15-04-05"
	updateInterval   = 24 * time.Hour
)

// SelfUpdater allows for running self-update of pmm-server-upgrade.
type SelfUpdater struct {
	l                *logrus.Entry
	apiServer        serverStartStopper
	disableImagePull bool
	docker           containerManager
	dockerImage      string
	triggerOnStart   bool
	updater          updateRunningChecker

	// isRunning is true if self-update is running.
	isRunning sync.Mutex
}

// New returns new SelfUpdater.
func New(
	docker containerManager, dockerImage string, disableImagePull bool,
	apiServer serverStartStopper, updater updateRunningChecker, triggerOnStart bool,
) *SelfUpdater {
	return &SelfUpdater{
		l:                logrus.WithField("component", "self-updater"),
		apiServer:        apiServer,
		disableImagePull: disableImagePull,
		docker:           docker,
		dockerImage:      dockerImage,
		triggerOnStart:   triggerOnStart,
		updater:          updater,
	}
}

// Start starts self-updater main loop.
func (s *SelfUpdater) Start(ctx context.Context) {
	go func() {
		for {
			if s.triggerOnStart {
				s.run(ctx)
			}

			s.l.Infof("Next check for updates scheduled in %s", updateInterval.String())

			select {
			case <-ctx.Done():
				return
			case <-time.After(updateInterval):
				s.run(ctx)
			}
		}
	}()
}

// run initiates check self-update.
func (s *SelfUpdater) run(ctx context.Context) {
	s.l.Info("Checking for updates to pmm-server-upgrade")
	if err := s.maybeUpdate(ctx); err != nil {
		s.l.Error(err)

		s.l.Info("Starting API server")
		// Starting API server gracefully returns if it's already running
		if updater := s.apiServer.Start(ctx); updater != nil {
			s.updater = updater
		}
	}
}

// maybeUpdate checks if update is available and starts self-update if it is.
func (s *SelfUpdater) maybeUpdate(ctx context.Context) error {
	if ok := s.isRunning.TryLock(); !ok {
		s.l.Warn("Self update is already running. Aborting")
		return nil
	}
	defer s.isRunning.Unlock()

	currentContainer, err := s.findSelfContainer(ctx)
	if err != nil {
		return err
	}

	s.l.Infof("Found this instance is running in container %s", currentContainer.ID[0:12])

	// Check if we shall update
	shallUpdate, err := s.prepareUpdate(ctx, currentContainer)
	if err != nil || !shallUpdate {
		return err
	}

	// Start new pmm-server-upgrade
	containerID, err := s.startNewContainer(ctx, currentContainer)
	if err != nil {
		if containerID != "" {
			// Stop the new container
			go func() {
				if err := s.stopContainerAndDisableRestart(ctx, containerID); err != nil {
					s.l.Error(err)
				}
			}()
		}
		return err
	}

	// Stop the current container
	if err = s.stopContainerAndDisableRestart(ctx, currentContainer.ID); err != nil {
		return err
	}

	return nil
}

func (s *SelfUpdater) findSelfContainer(ctx context.Context) (types.ContainerJSON, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return types.ContainerJSON{}, err
	}

	return s.docker.ContainerInspect(ctx, hostname)
}

func (s *SelfUpdater) startNewContainer(ctx context.Context, currentContainer types.ContainerJSON) (string, error) {
	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
		VolumesFrom:   []string{currentContainer.ID + ":rw"},
	}

	containerName := fmt.Sprintf("pmm-server-upgrade-%s", time.Now().Format(dateSuffixFormat))

	s.l.Infof("Starting new container %s", containerName)
	containerID, err := s.docker.RunContainer(ctx, &container.Config{
		Image: s.dockerImage,
		Env:   currentContainer.Config.Env,
		Cmd:   currentContainer.Config.Cmd,
	}, hostConfig, containerName)
	if err != nil {
		return containerID, err
	}

	s.l.Infof("Started container %s", containerID[0:12])

	// Check if new container is healthy
	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	s.l.Infof("Waiting for container %s to become healthy", containerID[0:12])
	healthy := <-s.docker.WaitForHealthyContainer(ctxTimeout, containerID)
	if healthy.Error != nil {
		return containerID, healthy.Error
	}
	s.l.Infof("Container %s is healthy", containerID[0:12])

	return containerID, nil
}

func (s *SelfUpdater) disableRestartPolicy(ctx context.Context, containerID string) error {
	s.l.Infof("Disabling restart policy on container %s", containerID[0:12])
	_, err := s.docker.ContainerUpdate(ctx, containerID, container.UpdateConfig{
		RestartPolicy: container.RestartPolicy{Name: "no"},
	})

	return err
}

func (s *SelfUpdater) downloadLatestImage(ctx context.Context) error {
	if s.disableImagePull {
		return nil
	}

	s.l.Infof("Downloading Docker image %s", s.dockerImage)
	r, err := s.docker.PullImage(ctx, s.dockerImage, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	_, err = io.Copy(os.Stdout, r)
	if err != nil {
		s.l.Error(err)
	}

	return nil
}

func (s *SelfUpdater) isUpdateAvailable(ctx context.Context, currentContainer types.ContainerJSON) (bool, error) {
	newImage, _, err := s.docker.ImageInspectWithRaw(ctx, s.dockerImage)
	if err != nil {
		return false, err
	}

	s.l.Debugf(
		"Docker image hash. Current = %s. New = %s",
		currentContainer.ContainerJSONBase.Image,
		newImage.ID)

	// Check if we're running an older version
	if currentContainer.ContainerJSONBase.Image == newImage.ID {
		return false, nil
	}

	return true, nil
}

func (s *SelfUpdater) prepareUpdate(ctx context.Context, currentContainer types.ContainerJSON) (bool, error) {
	// Download latest docker image
	if err := s.downloadLatestImage(ctx); err != nil {
		return false, err
	}

	// Check if newer version is available
	canUpdate, err := s.isUpdateAvailable(ctx, currentContainer)
	if err != nil {
		return false, err
	}

	if !canUpdate {
		s.l.Info("Already running the latest version.")
		return false, nil
	}

	s.l.Info("Newer version is available. Starting update")

	// Check if update is running
	if s.updater.IsAnyUpdateRunning() {
		s.l.Info("PMM Server update in progress. Aborting update")
		return false, nil
	}

	// Stop unix socket
	s.l.Infof("Stopping API server")
	s.apiServer.Stop()

	// Check if update is running, again to avoid race conditions
	if s.updater.IsAnyUpdateRunning() {
		s.l.Info("PMM Server update in progress. Aborting update")
		return false, nil
	}

	return true, nil
}

func (s *SelfUpdater) stopContainerAndDisableRestart(ctx context.Context, containerID string) error {
	if err := s.disableRestartPolicy(ctx, containerID); err != nil {
		s.l.Error(err)
	}

	timeout := 30 * time.Second

	s.l.Infof("Stopping container %s", containerID[0:12])
	err := s.docker.ContainerStop(ctx, containerID, &timeout)
	if err != nil {
		return err
	}

	return nil
}
