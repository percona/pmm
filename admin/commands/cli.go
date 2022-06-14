package commands

import (
	"time"
)

type StatusCmd struct {
	Timeout time.Duration `name:"wait" help:"Time to wait for a successful response from pmm-agent"`
}

type SummaryCmd struct {
	Filename   string `name:"filename" help:"Summary archive filename"`
	SkipServer bool   `name:"skip-server" help:"Skip fetching logs.zip from PMM Server"`
	Pprof      bool   `name:"pprof" help:"Include performance profiling data"`
}
