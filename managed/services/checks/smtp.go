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

package checks

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	gomail "gopkg.in/mail.v2"

	"github.com/percona/pmm/managed/services"
)

// smtpConfig holds the SMTP settings PMM reuses from the bundled Grafana. PMM Server configures
// Grafana's SMTP via GF_SMTP_* environment variables, and the pmm-managed process inherits them,
// so email delivery shares Grafana's single SMTP configuration instead of duplicating it.
type smtpConfig struct {
	enabled     bool
	host        string
	user        string
	password    string
	fromAddress string
	fromName    string
	skipVerify  bool
	startTLS    string
}

// smtpConfigFromEnv reads the Grafana SMTP settings from the inherited GF_SMTP_* environment.
func smtpConfigFromEnv() smtpConfig {
	enabled, _ := strconv.ParseBool(os.Getenv("GF_SMTP_ENABLED"))
	skipVerify, _ := strconv.ParseBool(os.Getenv("GF_SMTP_SKIP_VERIFY"))
	return smtpConfig{
		enabled:     enabled,
		host:        os.Getenv("GF_SMTP_HOST"),
		user:        os.Getenv("GF_SMTP_USER"),
		password:    os.Getenv("GF_SMTP_PASSWORD"),
		fromAddress: os.Getenv("GF_SMTP_FROM_ADDRESS"),
		fromName:    os.Getenv("GF_SMTP_FROM_NAME"),
		skipVerify:  skipVerify,
		startTLS:    os.Getenv("GF_SMTP_STARTTLS_POLICY"),
	}
}

// dialer builds a gomail dialer from the config, mirroring Grafana's createDialer
// (grafana pkg/services/notifications/smtp.go).
func (c smtpConfig) dialer() (*gomail.Dialer, error) {
	host, portStr, err := net.SplitHostPort(c.host)
	if err != nil {
		return nil, fmt.Errorf("invalid GF_SMTP_HOST %q: %w", c.host, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP port in %q: %w", c.host, err)
	}

	d := gomail.NewDialer(host, port, c.user, c.password)
	d.TLSConfig = &tls.Config{
		ServerName: host,
		// Honors GF_SMTP_SKIP_VERIFY, matching Grafana's own SMTP behavior.
		InsecureSkipVerify: c.skipVerify, //nolint:gosec
	}
	switch c.startTLS {
	case "NoStartTLS":
		d.StartTLSPolicy = gomail.NoStartTLS
	case "MandatoryStartTLS":
		d.StartTLSPolicy = gomail.MandatoryStartTLS
	default:
		d.StartTLSPolicy = gomail.OpportunisticStartTLS
	}

	return d, nil
}

// sendAdvisorEmail emails a pre-composed report to the recipients using the Grafana-configured
// SMTP server.
func (s *Service) sendAdvisorEmail(to []string, subject, body string) error {
	if len(to) == 0 {
		return errors.New("no recipient addresses configured")
	}

	cfg := smtpConfigFromEnv()
	if !cfg.enabled {
		return fmt.Errorf("%w: GF_SMTP_ENABLED is not set", services.ErrSMTPNotConfigured)
	}
	if cfg.fromAddress == "" {
		return fmt.Errorf("%w: GF_SMTP_FROM_ADDRESS is empty", services.ErrSMTPNotConfigured)
	}

	return s.sendEmail(cfg, to, subject, body)
}

// sendEmail sends a plain-text email to the recipients using the Grafana-configured SMTP server.
func (s *Service) sendEmail(cfg smtpConfig, to []string, subject, body string) error {
	dialer, err := cfg.dialer()
	if err != nil {
		return err
	}

	m := gomail.NewMessage()
	if cfg.fromName != "" {
		m.SetAddressHeader("From", cfg.fromAddress, cfg.fromName)
	} else {
		m.SetHeader("From", cfg.fromAddress)
	}
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	return dialer.DialAndSend(m)
}
