package service

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"
)

func isSupervisord() bool {
	if _, err := os.Stat("/usr/bin/supervisord"); err == nil {
		return true
	}
	return false
}

type supervisord struct {
	i Interface
	*Config
}

func newSupervisordService(i Interface, c *Config) (Service, error) {
	s := &supervisord{
		i:      i,
		Config: c,
	}

	return s, nil
}

func (s *supervisord) String() string {
	if len(s.DisplayName) > 0 {
		return s.DisplayName
	}
	return s.Name
}

var errNoUserServiceSupervisord = errors.New("User services are not supported on Supervisord.")

func (s *supervisord) configPath() (cp string, err error) {
	if s.Option.bool(optionUserService, optionUserServiceDefault) {
		err = errNoUserServiceSupervisord
		return
	}
	cp = "/etc/supervisord.d/" + s.Config.Name + ".ini"
	return
}
func (s *supervisord) template() *template.Template {
	return template.Must(template.New("").Funcs(tf).Parse(supervisordScript))
}

func (s *supervisord) Install() error {
	confPath, err := s.configPath()
	if err != nil {
		return err
	}
	_, err = os.Stat(confPath)
	if err == nil {
		return fmt.Errorf("Init already exists: %s", confPath)
	}

	f, err := os.Create(confPath)
	if err != nil {
		return err
	}
	defer f.Close()

	path, err := s.execPath()
	if err != nil {
		return err
	}

	var to = &struct {
		*Config
		Path string
	}{
		s.Config,
		path,
	}

	return s.template().Execute(f, to)
}

func (s *supervisord) Uninstall() error {
	cp, err := s.configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(cp); err != nil {
		return err
	}
	// Remove log file.
	os.Remove(fmt.Sprintf("/var/log/%s.log", s.Name))
	return nil
}

func (s *supervisord) Logger(errs chan<- error) (Logger, error) {
	if system.Interactive() {
		return ConsoleLogger, nil
	}
	return s.SystemLogger(errs)
}
func (s *supervisord) SystemLogger(errs chan<- error) (Logger, error) {
	return newSysLogger(s.Name, errs)
}

func (s *supervisord) Run() (err error) {
	err = s.i.Start(s)
	if err != nil {
		return err
	}

	s.Option.funcSingle(optionRunWait, func() {
		var sigChan = make(chan os.Signal, 3)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
	})()

	return s.i.Stop(s)
}

func (s *supervisord) Start() error {
	if err := run("supervisorctl", "reread"); err != nil {
		return err
	}
	return run("supervisorctl", "add", s.Name)
}

func (s *supervisord) Stop() error {
	run("supervisorctl", "stop", s.Name) // ignore error
	return run("supervisorctl", "remove", s.Name)
}

func (s *supervisord) Restart() error {
	// we do not use `supervisorctl restart` because we want Start() to call reread

	if err := s.Stop(); err != nil {
		return err
	}

	time.Sleep(50 * time.Millisecond)

	return s.Start()
}

func (s *supervisord) Status() error {
	// supervisorctl status does not return non-zero code when service is stopped.
	cmd := fmt.Sprintf("supervisorctl status %s | grep RUNNING", s.Name)
	return run("/bin/sh", "-c", cmd)
}

const supervisordScript = `# {{.Description}}

[program:{{.Name}}]
command = {{.Path}}{{range .Arguments}} {{.}}{{end}}

{{if .Environment}}environment {{range .Environment}}{{.|envKey}}={{.|envValue|cmd}} {{end}}{{end}}

{{if .UserName}}user {{.UserName}}{{end}}

{{if .WorkingDirectory}}directory {{.WorkingDirectory}}{{end}}

autorestart = true

stdout_logfile = /var/log/{{.Name}}.log
stderr_logfile = /var/log/{{.Name}}.log
`
