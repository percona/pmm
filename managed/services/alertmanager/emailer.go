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

package alertmanager

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify/email"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

// Emailer is responsible for sending emails using alertmanger Email notifier.
type Emailer struct {
	l *logrus.Logger
}

// NewEmailer creates Emailer instance.
func NewEmailer(l *logrus.Logger) *Emailer {
	return &Emailer{l: l}
}

type loggerFunc func(level logrus.Level, args ...interface{})

// Log performs logging operation to logrus logger.
func (f loggerFunc) Log(values ...interface{}) error {
	f(logrus.DebugLevel, values)
	return nil
}

// Send sends an email to `emailTo` recipient using given settings.
func (e *Emailer) Send(ctx context.Context, settings *models.EmailAlertingSettings, emailTo string) error {
	host, port, err := net.SplitHostPort(settings.Smarthost)
	if err != nil {
		return models.NewInvalidArgumentError("invalid smarthost: %q", err.Error())
	}

	if port == "" {
		return models.NewInvalidArgumentError("address %q: port cannot be empty", port)
	}

	emailConfig := &config.EmailConfig{
		NotifierConfig: config.NotifierConfig{},
		To:             emailTo,
		From:           settings.From,
		Hello:          settings.Hello,
		Smarthost: config.HostPort{
			Host: host,
			Port: port,
		},
		AuthUsername: settings.Username,
		AuthPassword: config.Secret(settings.Password),
		AuthSecret:   config.Secret(settings.Secret),
		AuthIdentity: settings.Identity,
		Headers: map[string]string{
			"Subject": `Test alert.`,
		},
		HTML:       emailTemplate,
		RequireTLS: &settings.RequireTLS,
	}

	tmpl, err := template.FromGlobs([]string{"*"})
	if err != nil {
		return err
	}
	tmpl.ExternalURL, err = url.Parse("https://example.com")
	if err != nil {
		return err
	}

	alertmanagerEmail := email.New(emailConfig, tmpl, loggerFunc(e.l.Log))
	if _, err := alertmanagerEmail.Notify(ctx, &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				model.AlertNameLabel: model.LabelValue(fmt.Sprintf("Test alert %s", time.Now().String())),
				"severity":           "notice",
			},
			Annotations: model.LabelSet{
				"summary":     "This is a test alert.",
				"description": "Long description.",
				"rule":        "example-violated-rule",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Minute),
		},
		Timeout: true,
	}); err != nil {
		return models.NewInvalidArgumentError(err.Error())
	}

	return nil
}
