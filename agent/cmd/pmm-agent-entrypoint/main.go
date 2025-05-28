// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package main provides the entry point for the PMM Agent.
package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/pkg/errors"
	reaper "github.com/ramr/go-reaper"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/utils/logger"
)

var helpText = `
PMM 2.x Client Docker container.

It runs pmm-agent as a process with PID 1.
It is configured entirely by environment variables. Arguments or flags are not used.

The following environment variables are recognized by the Docker entrypoint:
* PMM_AGENT_SETUP            - if true, 'pmm-agent setup' is called before 'pmm-agent run'.
* PMM_AGENT_PRERUN_FILE      - if non-empty, runs given file with 'pmm-agent run' running in the background.
* PMM_AGENT_PRERUN_SCRIPT    - if non-empty, runs given shell script content with 'pmm-agent run' running in the background.
* PMM_AGENT_SIDECAR          - if true, 'pmm-agent' will be restarted in case of it's failed.
* PMM_AGENT_SIDECAR_SLEEP    - time to wait before restarting pmm-agent if PMM_AGENT_SIDECAR is true. 1 second by default.

Additionally, the many environment variables are recognized by pmm-agent itself.
The following help text shows them as [PMM_AGENT_XXX].
`

type restartPolicy int

const (
	doNotRestart restartPolicy = iota + 1
	restartAlways
	restartOnFail
)

var (
	pmmAgentSetup = kingpin.Flag("pmm-agent-setup",
		"if true, 'pmm-agent setup' is called before 'pmm-agent run'").Default("false").Envar("PMM_AGENT_SETUP").Bool()
	pmmAgentSidecar = kingpin.Flag("pmm-agent-sidecar",
		"if true, 'pmm-agent' will be restarted in case of it's failed").Default("false").Envar("PMM_AGENT_SIDECAR").Bool()
	pmmAgentSidecarSleep = kingpin.Flag("pmm-agent-sidecar-sleep",
		"time to wait before restarting pmm-agent if PMM_AGENT_SIDECAR is true. 1 second by default").Default("1").Envar("PMM_AGENT_SIDECAR_SLEEP").Int()
	pmmAgentPrerunFile = kingpin.Flag("pmm-agent-prerun-file",
		"if non-empty, runs given file with 'pmm-agent run' running in the background").Envar("PMM_AGENT_PRERUN_FILE").String()
	pmmAgentPrerunScript = kingpin.Flag("pmm-agent-prerun-script",
		"if non-empty, runs given shell script content with 'pmm-agent run' running in the background").Envar("PMM_AGENT_PRERUN_SCRIPT").String()
)

var pmmAgentProcessID = 0

// isAgentConfigured checks if pmm-agent is already configured by checking if it has an ID.
func isAgentConfigured(l *logrus.Entry) bool {
	// Try to load the configuration to check if agent has an ID
	configStorage := config.NewStorage(nil)
	_, err := configStorage.Reload(l)

	// If there's an error loading config (e.g., file doesn't exist), agent is not configured
	var e config.ConfigFileDoesNotExistError
	if err != nil && errors.As(err, &e) {
		return false
	}
	if err != nil {
		l.Debugf("Error loading configuration: %s", err)
		return false
	}

	cfg := configStorage.Get()
	// Agent is configured if it has an ID
	return cfg.ID != ""
}

func runPmmAgent(ctx context.Context, commandLineArgs []string, restartPolicy restartPolicy, l *logrus.Entry, pmmAgentSidecarSleep int) int {
	pmmAgentFullCommand := "pmm-agent " + strings.Join(commandLineArgs, " ")
	for {
		select {
		case <-ctx.Done():
			return 1
		default:
		}
		var exitCode int
		l.Infof("Starting 'pmm-agent %s'...", strings.Join(commandLineArgs, " "))
		cmd := commandPmmAgent(commandLineArgs)
		if err := cmd.Start(); err != nil {
			l.Errorf("Can't run: '%s', Error: %s", commandLineArgs, err)
			exitCode = -1
		} else {
			pmmAgentProcessID = cmd.Process.Pid
			if err := cmd.Wait(); err != nil {
				exitError, ok := err.(*exec.ExitError) //nolint:errorlint
				if !ok {
					l.Errorf("Can't get exit code for '%s'. err: %s", pmmAgentFullCommand, err)
					exitCode = -1
				} else {
					exitCode = exitError.ExitCode()
				}
			}
		}
		l.Infof("'%s' exited with %d", pmmAgentFullCommand, exitCode)

		if restartPolicy == restartAlways || (restartPolicy == restartOnFail && exitCode != 0) {
			l.Infof("Restarting `%s` in %d seconds because PMM_AGENT_SIDECAR is enabled...", pmmAgentFullCommand, pmmAgentSidecarSleep)
			time.Sleep(time.Duration(pmmAgentSidecarSleep) * time.Second)
		} else {
			return exitCode
		}
	}
}

func commandPmmAgent(args []string) *exec.Cmd {
	const pmmAgentCommandName = "pmm-agent"
	command := exec.Command(pmmAgentCommandName, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command
}

func sendSIGKILLwithTimeout(process *os.Process, timeout int, l *logrus.Entry) *time.Timer {
	return time.AfterFunc(time.Second*time.Duration(timeout), func() {
		l.Infof("Failed to finish process in %d second. Send SIGKILL", timeout)
		err := process.Kill()
		if err != nil {
			l.Warnf("Failed to kill pmm-agent: %s", err)
		}
	})
}

func main() {
	go reaper.Reap()
	kingpin.Parse()

	var status int

	logger.SetupGlobalLogger()

	l := logrus.WithField("component", "entrypoint")

	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s := <-signals
		signal.Stop(signals)
		l.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
		if pmmAgentProcessID != 0 {
			l.Info("Graceful shutdown for pmm-agent...")
			// graceful shutdown for pmm-agent
			if err := syscall.Kill(pmmAgentProcessID, syscall.SIGTERM); err != nil {
				l.Warn("Failed to send SIGTERM, command must have exited:", err)
			}
			pmmAgentProcess, _ := os.FindProcess(pmmAgentProcessID) // always succeeds even process is not exist
			preSIGKILLtimeout := 10
			timer := sendSIGKILLwithTimeout(pmmAgentProcess, preSIGKILLtimeout, l)
			_, err := pmmAgentProcess.Wait()
			if err != nil {
				l.Warn("Failed to finish pmm-agent")
			}
			timer.Stop()
		}
		cancel()
		os.Exit(1)
	}()

	if len(os.Args) > 1 {
		l.Info(helpText)
		exec.CommandContext(ctx, "pmm-agent", "setup", "--help")
		os.Exit(1)
	}

	l.Infof("Run setup: %t Sidecar mode: %t", *pmmAgentSetup, *pmmAgentSidecar)
	if *pmmAgentPrerunFile != "" && *pmmAgentPrerunScript != "" {
		l.Error("Both PMM_AGENT_PRERUN_FILE and PMM_AGENT_PRERUN_SCRIPT cannot be set.")
		os.Exit(1)
	}

	if *pmmAgentSetup { //nolint:nestif
		// Check if agent is already configured
		if isAgentConfigured(l) {
			l.Info("PMM agent is already configured, skipping setup")
		} else {
			var agent *exec.Cmd
			restartPolicy := doNotRestart
			if *pmmAgentSidecar {
				restartPolicy = restartOnFail
				l.Info("Starting pmm-agent for liveness probe...")
				agent = commandPmmAgent([]string{"run"})
				err := agent.Start()
				if err != nil {
					l.Fatalf("Can't run pmm-agent: %s", err)
				}
			}
			statusSetup := runPmmAgent(ctx, []string{"setup"}, restartPolicy, l, *pmmAgentSidecarSleep)
			if statusSetup != 0 {
				os.Exit(statusSetup)
			}
			if *pmmAgentSidecar {
				l.Info("Stopping pmm-agent...")
				if err := agent.Process.Signal(syscall.SIGTERM); err != nil {
					l.Fatal("Failed to kill pmm-agent: ", err)
				}
			}
		}
	}

	status = 0
	if *pmmAgentPrerunFile != "" || *pmmAgentPrerunScript != "" { //nolint:nestif
		l.Info("Starting pmm-agent for prerun...")
		agent := commandPmmAgent([]string{"run"})
		err := agent.Start()
		if err != nil {
			l.Errorf("Failed to run pmm-agent run command: %s", err)
		}

		if *pmmAgentPrerunFile != "" {
			l.Infof("Running prerun file %s...", *pmmAgentPrerunFile)
			cmd := exec.CommandContext(ctx, *pmmAgentPrerunFile) //nolint:gosec
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				if exitError, ok := err.(*exec.ExitError); ok { //nolint:errorlint
					status = exitError.ExitCode()
					l.Infof("Prerun file exited with %d", exitError.ExitCode())
				}
			}
		}

		if *pmmAgentPrerunScript != "" {
			l.Infof("Running prerun shell script %s...", *pmmAgentPrerunScript)
			cmd := exec.CommandContext(ctx, "/bin/sh", "-c", *pmmAgentPrerunScript) //nolint:gosec
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				var exitError *exec.ExitError
				if errors.As(err, &exitError) {
					status = exitError.ExitCode()
					l.Infof("Prerun shell script exited with %d", exitError.ExitCode())
				}
			}
		}

		l.Info("Stopping pmm-agent...")
		if err := agent.Process.Signal(syscall.SIGTERM); err != nil {
			l.Infof("Failed to term pmm-agent: %s", err)
		}

		// kill pmm-agent process in 10 seconds if SIGTERM doesn't work
		preSIGKILLtimeout := 10
		timer := sendSIGKILLwithTimeout(agent.Process, preSIGKILLtimeout, l)

		err = agent.Wait()
		if err != nil {
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				status = exitError.ExitCode()
				l.Infof("Prerun pmm-agent exited with %d", exitError.ExitCode())
			} else {
				l.Warnf("Can't get exit code for pmm-agent. Error code: %s", err)
			}
		}
		timer.Stop()

		if status != 0 && !*pmmAgentSidecar {
			os.Exit(status)
		}
	}
	restartPolicy := doNotRestart
	if *pmmAgentSidecar {
		restartPolicy = restartAlways
	}
	runPmmAgent(ctx, []string{"run"}, restartPolicy, l, *pmmAgentSidecarSleep)
}
