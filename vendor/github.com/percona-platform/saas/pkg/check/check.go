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
			if err := check.validate(); err != nil {
				if params.DisallowInvalidChecks {
					return nil, err
				}

				continue // skip invalid check
			}

			res = append(res, check)
		}
	}
}

// Type represents check type.
type Type string

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

// Tier represents check tier.
type Tier string

// Supported check tiers.
const (
	Anonymous  = Tier("anonymous")
	Registered = Tier("registered")
)

// Validate validates tier value.
func (t Tier) Validate() error {
	switch t {
	case Anonymous:
	case Registered:
	default:
		return errors.Errorf("unknown check tier: %q", t)
	}

	return nil
}

// Check represents security check structure.
type Check struct {
	Version uint32 `yaml:"version"`
	Name    string `yaml:"name"`
	Tiers   []Tier `yaml:"tiers,flow,omitempty"`
	Type    Type   `yaml:"type"`
	Query   string `yaml:"query,omitempty"`
	Script  string `yaml:"script"`
}

// the same as Prometheus label format
//nolint:gochecknoglobals
var nameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// validate validates check for minimal correctness.
func (c *Check) validate() error {
	if c.Version != 1 {
		return errors.Errorf("unexpected version %d", c.Version)
	}

	if !nameRE.MatchString(c.Name) {
		return errors.New("invalid check name")
	}

	if err := c.validateTiers(); err != nil {
		return err
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
		fallthrough
	case MongoDBGetCmdLineOpts:
		return nil
	case "":
		return errors.New("check type is empty")
	default:
		return errors.Errorf("unknown check type: %s", c.Type)
	}
}

// validateTiers validates tiers field if it's present.
func (c *Check) validateTiers() error {
	m := make(map[Tier]struct{}, len(c.Tiers))
	for _, tier := range c.Tiers {
		if err := tier.Validate(); err != nil {
			return err
		}

		if _, ok := m[tier]; ok {
			return errors.Errorf("duplicate tier: %q", tier)
		}
		m[tier] = struct{}{}
	}

	return nil
}
