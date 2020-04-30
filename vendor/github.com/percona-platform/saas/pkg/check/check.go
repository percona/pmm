// Package check implements checks parsing and validating.
package check

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Verify checks signature of passed data with provided public key and
// returns error in case of any problem.
func Verify(data []byte, publicKey, sig string) error {
	lines := strings.SplitN(sig, "\n", 4)
	if len(lines) < 4 {
		return errors.New("incomplete signature")
	}

	sBin, err := base64.StdEncoding.DecodeString(lines[1])
	if err != nil || len(sBin) != 74 {
		return errors.New("invalid signature")
	}
	gBin, err := base64.StdEncoding.DecodeString(lines[3])
	if err != nil || len(gBin) != 64 {
		return errors.New("invalid global signature")
	}
	kBin, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil || len(kBin) != 42 {
		return errors.New("invalid public key")
	}

	sAlg, sKeyID, sSig := sBin[0:2], sBin[2:10], sBin[10:74]
	kAlg, kKeyID, kKey := kBin[0:2], kBin[2:10], kBin[10:42]

	if !bytes.Equal(kAlg, sAlg) {
		return errors.New("incompatible signature algorithm")
	}
	if sAlg[0] != 0x45 || sAlg[1] != 0x64 {
		return errors.New("unsupported signature algorithm")
	}
	if !bytes.Equal(kKeyID, sKeyID) {
		return errors.New("incompatible key identifiers")
	}
	if !strings.HasPrefix(lines[2], "trusted comment: ") {
		return errors.New("unexpected format for the trusted comment")
	}
	if !ed25519.Verify(ed25519.PublicKey(kKey), data, sSig) {
		return errors.New("invalid signature")
	}
	if !ed25519.Verify(ed25519.PublicKey(kKey), append(sSig, []byte(lines[2])[17:]...), gBin) {
		return errors.New("invalid global signature")
	}
	return nil
}

// Parse returns a slice of validated checks parsed from YAML passed via a reader.
// It can handle multi-document YAMLs: parsing result will be a single slice
// that contains checks form every parsed document.
func Parse(reader io.Reader) ([]Check, error) {
	d := yaml.NewDecoder(reader)

	type checks struct {
		Checks []Check `yaml:"checks"`
	}

	var res []Check
	for {
		var c checks
		if err := d.Decode(&c); err != nil {
			if err == io.EOF {
				return res, nil
			}
			return nil, errors.Wrap(err, "failed to parse checks")
		}

		for _, check := range c.Checks {
			if err := check.validate(); err != nil {
				return nil, err
			}
			res = append(res, check)
		}
	}
}

// Type represents check type.
type Type string

// Supported check types.
const (
	MySQLShow           = Type("MYSQL_SHOW")
	MySQLSelect         = Type("MYSQL_SELECT")
	PostgreSQLShow      = Type("POSTGRESQL_SHOW")
	PostgreSQLSelect    = Type("POSTGRESQL_SELECT")
	MongoDBGetParameter = Type("MONGODB_GETPARAMETER")
	MongoDBBuildInfo    = Type("MONGODB_BUILDINFO")
)

// Check represents security check structure.
type Check struct {
	Name    string `yaml:"name"`
	Version uint32 `yaml:"version"`
	Type    Type   `yaml:"type"`
	Query   string `yaml:"query"`
	Script  string `yaml:"script"`
}

// the same as Prometheus label format
//nolint:gochecknoglobals
var nameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// validate validates check for minimal correctness.
func (c *Check) validate() error {
	if !nameRE.MatchString(c.Name) {
		return errors.New("invalid check name")
	}

	if err := c.validateType(); err != nil {
		return err
	}

	if err := c.validateQuery(); err != nil {
		return err
	}

	if err := c.validateScript(); err != nil {
		return err
	}

	if c.Script == "" {
		return errors.New("check script is empty")
	}

	return nil
}

func (c *Check) validateScript() error {
	if strings.ContainsRune(c.Script, '\t') {
		return errors.New("script should use spaces for indentation, not tabs")
	}

	return nil
}

func (c *Check) validateQuery() error {
	switch c.Type {
	case PostgreSQLShow:
		fallthrough
	case MongoDBGetParameter:
		fallthrough
	case MongoDBBuildInfo:
		if c.Query != "" {
			return errors.Errorf("%s check type should have empty query", c.Type)
		}
	default:
		if c.Query == "" {
			return errors.New("check query is empty")
		}
	}

	return nil
}

// validateType validates check type.
func (c *Check) validateType() error {
	switch c.Type {
	case MySQLShow:
		fallthrough
	case MySQLSelect:
		fallthrough
	case PostgreSQLShow:
		fallthrough
	case PostgreSQLSelect:
		fallthrough
	case MongoDBGetParameter:
		fallthrough
	case MongoDBBuildInfo:
		return nil
	case "":
		return errors.New("check type is empty")
	default:
		return errors.Errorf("unknown check type: %s", c.Type)
	}
}

//go:generate ../../bin/stringer -type=Severity -linecomment

// Severity represents severity level.
type Severity int

// Supported severity levels.
const (
	Unknown   Severity = iota // unknown
	Emergency                 // emergency
	Alert                     // alert
	Critical                  // critical
	Error                     // error
	Warning                   // warning
	Notice                    // notice
	Info                      // info
	Debug                     // debug
)

// StrToSeverity casts string to Severity.
func StrToSeverity(s string) Severity {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "emergency":
		return Emergency
	case "alert":
		return Alert
	case "critical":
		return Critical
	case "error":
		return Error
	case "warning":
		return Warning
	case "notice":
		return Notice
	case "info":
		return Info
	case "debug":
		return Debug
	default:
		return Unknown
	}
}

// Result represents a single check script result that is used to generate alert.
type Result struct {
	Summary     string
	Description string
	Severity    Severity
	Labels      map[string]string
}

// Validate validates check result for minimal correctness.
func (r *Result) Validate() error {
	if err := r.validateSeverity(); err != nil {
		return err
	}

	if r.Summary == "" {
		return errors.New("summary is empty")
	}

	return nil
}

// validateSeverity validates check result severity level.
func (r *Result) validateSeverity() error {
	if r.Severity < Emergency || r.Severity > Debug {
		return errors.Errorf("unknown result severity: %s", r.Severity)
	}

	return nil
}
