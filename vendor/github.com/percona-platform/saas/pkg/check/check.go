// Package check implements checks parsing and validation.
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

	"github.com/percona-platform/saas/pkg/common"
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

// ParseParams represents optional Parse function parameters.
type ParseParams struct {
	DisallowUnknownFields bool // if true, return errors for unexpected YAML fields
	DisallowInvalidChecks bool // if true, return errors for invalid checks instead of skipping them
}

// Parse returns a slice of validated checks parsed from YAML passed via a reader.
// It can handle multi-document YAMLs: parsing result will be a single slice
// that contains checks form every parsed document.
func Parse(reader io.Reader, params *ParseParams) ([]Check, error) {
	if params == nil {
		params = new(ParseParams)
	}

	d := yaml.NewDecoder(reader)
	d.KnownFields(params.DisallowUnknownFields)

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
			if err := check.Validate(); err != nil {
				if params.DisallowInvalidChecks {
					return nil, err
				}

				continue // skip invalid check
			}

			res = append(res, check)
		}
	}
}

// Supported check types.
const (
	MySQLShow             = Type("MYSQL_SHOW")
	MySQLSelect           = Type("MYSQL_SELECT")
	PostgreSQLShow        = Type("POSTGRESQL_SHOW")
	PostgreSQLSelect      = Type("POSTGRESQL_SELECT")
	MongoDBGetParameter   = Type("MONGODB_GETPARAMETER")
	MongoDBBuildInfo      = Type("MONGODB_BUILDINFO")
	MongoDBGetCmdLineOpts = Type("MONGODB_GETCMDLINEOPTS")
)

// Type represents check type.
type Type string

// Validate validates check type.
func (t Type) Validate() error {
	switch t {
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
		fallthrough
	case MongoDBGetCmdLineOpts:
		return nil
	case "":
		return errors.New("check type is empty")
	default:
		return errors.Errorf("unknown check type: %s", t)
	}
}

// Supported check intervals.
const (
	Standard = Interval("standard")
	Frequent = Interval("frequent")
	Rare     = Interval("rare")
)

// Interval represents check execution interval.
type Interval string

// Validate validates check interval.
func (i Interval) Validate() error {
	switch i {
	case Standard:
		fallthrough
	case Frequent:
		fallthrough
	case Rare:
		fallthrough
	case "":
		return nil
	default:
		return errors.Errorf("unknown check interval: %s", i)
	}
}

// Check represents security check structure.
type Check struct {
	Version     uint32        `yaml:"version"`
	Name        string        `yaml:"name"`
	Summary     string        `yaml:"summary"`
	Description string        `yaml:"description"`
	Type        Type          `yaml:"type"`
	Tiers       []common.Tier `yaml:"tiers,flow,omitempty"`
	Interval    Interval      `yaml:"interval,omitempty"`
	Query       string        `yaml:"query,omitempty"`
	Script      string        `yaml:"script"`
}

// The same as Prometheus label format.
var nameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Validate validates check for minimal correctness.
func (c *Check) Validate() error {
	var err error
	if c.Version != 1 {
		return errors.Errorf("unexpected version %d", c.Version)
	}

	if !nameRE.MatchString(c.Name) {
		return errors.New("invalid check name")
	}

	if err = common.ValidateTiers(c.Tiers); err != nil {
		return err
	}

	if err = c.Interval.Validate(); err != nil {
		return err
	}

	if err = c.Type.Validate(); err != nil {
		return err
	}

	if err = c.validateQuery(); err != nil {
		return err
	}

	if err = c.validateScript(); err != nil {
		return err
	}

	if c.Script == "" {
		return errors.New("check script is empty")
	}

	if c.Summary == "" {
		return errors.New("summary is empty")
	}

	if c.Description == "" {
		return errors.New("description is empty")
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
		fallthrough
	case MongoDBGetCmdLineOpts:
		if c.Query != "" {
			return errors.Errorf("%s check type should have empty query", c.Type)
		}
	case PostgreSQLSelect:
		fallthrough
	case MySQLShow:
		fallthrough
	case MySQLSelect:
		if c.Query == "" {
			return errors.New("check query is empty")
		}
	}

	return nil
}
