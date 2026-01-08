// Package check implements checks parsing and validation.
package check

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"gopkg.in/yaml.v3"
)

// The same as Prometheus label format.
var nameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Verify checks signature of passed data with provided public key and
// returns error in case of any problem.
func Verify(data []byte, publicKey, sig string) error { //nolint: cyclop
	lines := strings.SplitN(sig, "\n", 4) //nolint:mnd
	if len(lines) < 4 {                   //nolint:mnd
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

	// Key algorithm should be `Ed`.
	if kAlg[0] != 0x45 || kAlg[1] != 0x64 {
		return errors.New("unsupported key algorithm")
	}
	// Signature algorithm can be `Ed`(legacy) or `ED`(pre-hashed).
	if sAlg[0] != 0x45 || (sAlg[1] != 0x64 && sAlg[1] != 0x44) {
		return errors.New("unsupported signature algorithm")
	}

	// For pre-hashed signature get data hash.
	if sAlg[1] == 0x44 { //nolint:mnd
		h, _ := blake2b.New512(nil)
		h.Write(data)
		data = h.Sum(nil)
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
// that contains checks from every parsed document.
// Deprecated: use ParseChecks instead.
func Parse(reader io.Reader, params *ParseParams) ([]Check, error) {
	return ParseChecks(reader, params)
}

// ParseChecks returns a slice of validated checks parsed from YAML passed via a reader.
// It can handle multi-document YAMLs: parsing result will be a single slice
// that contains checks from every parsed document.
func ParseChecks(reader io.Reader, params *ParseParams) ([]Check, error) {
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
		if err := d.Decode(&c); err != nil { //nolint:musttag
			if errors.Is(err, io.EOF) {
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

// Supported query types.
const (
	MySQLShow                = Type("MYSQL_SHOW")
	MySQLSelect              = Type("MYSQL_SELECT")
	PostgreSQLShow           = Type("POSTGRESQL_SHOW")
	PostgreSQLSelect         = Type("POSTGRESQL_SELECT")
	MongoDBGetParameter      = Type("MONGODB_GETPARAMETER")
	MongoDBBuildInfo         = Type("MONGODB_BUILDINFO")
	MongoDBGetCmdLineOpts    = Type("MONGODB_GETCMDLINEOPTS")
	MongoDBReplSetGetStatus  = Type("MONGODB_REPLSETGETSTATUS")
	MongoDBGetDiagnosticData = Type("MONGODB_GETDIAGNOSTICDATA")
	MetricsInstant           = Type("METRICS_INSTANT")
	MetricsRange             = Type("METRICS_RANGE")
	ClickHouseSelect         = Type("CLICKHOUSE_SELECT")
)

// Type represents query type.
type Type string

// Validate validates query type.
func (t Type) Validate() error {
	switch t {
	case MySQLShow, MySQLSelect, PostgreSQLShow, PostgreSQLSelect,
		MongoDBGetParameter, MongoDBBuildInfo, MongoDBGetCmdLineOpts, MongoDBReplSetGetStatus,
		MongoDBGetDiagnosticData, ClickHouseSelect, MetricsInstant, MetricsRange:
		return nil
	case "":
		return errors.New("check type is empty")
	default:
		return errors.Errorf("unknown check type: %s", t)
	}
}

func isTypeSupportedByV1(t Type) bool {
	switch t { //nolint:exhaustive
	case MySQLShow, MySQLSelect, PostgreSQLShow, PostgreSQLSelect, MongoDBGetParameter,
		MongoDBBuildInfo, MongoDBGetCmdLineOpts, MongoDBReplSetGetStatus, MongoDBGetDiagnosticData:
		return true
	default:
		return false
	}
}

// Supported DB families.
const (
	MySQL      = Family("MYSQL")
	PostgreSQL = Family("POSTGRESQL")
	MongoDB    = Family("MONGODB")
)

// Family represents monitored service family.
type Family string

// Validate validates check family.
func (f Family) Validate() error {
	switch f {
	case MySQL:
		fallthrough
	case PostgreSQL:
		fallthrough
	case MongoDB:
		return nil
	case "":
		return errors.New("check family is empty")
	default:
		return errors.Errorf("unknown check family: %s", f)
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

// Parameter represents query parameter.
type Parameter string

// Available query parameters.
const (
	Lookback = Parameter("lookback")
	Range    = Parameter("range")
	Step     = Parameter("step")
	AllDBs   = Parameter("all_dbs")
)

// Query represents DB query of specified type.
type Query struct {
	Query      string
	Type       Type
	Parameters map[Parameter]string
}

// Validate validates query.
func (q Query) Validate() error {
	if err := q.Type.Validate(); err != nil {
		return err
	}

	if err := validateQuery(q.Type, q.Query); err != nil {
		return err
	}

	return validateQueryParameters(q.Type, q.Parameters)
}

// Check represents advisor check structure. Fields marked with v1 should not be used for version 2, and vice versa.
type Check struct {
	Version     uint32   `yaml:"version"`
	Name        string   `yaml:"name"`
	Summary     string   `yaml:"summary"`
	Description string   `yaml:"description"`
	Advisor     string   `yaml:"advisor"`
	Category    string   `yaml:"category,omitempty"` // deprecated
	Type        Type     `yaml:"type,omitempty"`     // for v1
	Family      Family   `yaml:"family,omitempty"`   // for v2, emulated via GetFamily for v1
	Interval    Interval `yaml:"interval,omitempty"`
	Query       string   `yaml:"query,omitempty"`   // for v1
	Queries     []Query  `yaml:"queries,omitempty"` // for v2
	Script      string   `yaml:"script"`
}

// GetFamily returns check family for both V1 and V2 check formats.
func (c *Check) GetFamily() Family {
	switch c.Version {
	case 1:
		switch c.Type {
		case MySQLSelect, MySQLShow:
			return MySQL

		case PostgreSQLSelect, PostgreSQLShow:
			return PostgreSQL

		case MongoDBGetParameter, MongoDBBuildInfo, MongoDBGetCmdLineOpts,
			MongoDBReplSetGetStatus, MongoDBGetDiagnosticData:
			return MongoDB

		case MetricsInstant, MetricsRange, ClickHouseSelect:
			return "" // Unsupported query types for V1, check is invalid
		}
	case 2: //nolint:mnd
		return c.Family
	}

	return ""
}

// Validate validates check for minimal correctness.
func (c *Check) Validate() error { //nolint: cyclop
	var err error

	if !nameRE.MatchString(c.Name) {
		return errors.New("invalid check name")
	}

	if err = c.Interval.Validate(); err != nil {
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

	if c.Advisor == "" {
		return errors.New("advisor name is missing")
	}

	switch c.Version {
	case 1:
		return c.validateV1()
	case 2: //nolint:mnd
		return c.validateV2()
	default:
		return errors.Errorf("unexpected version %d", c.Version)
	}
}

func (c *Check) validateV1() error {
	var err error
	if err = c.Type.Validate(); err != nil {
		return err
	}

	if !isTypeSupportedByV1(c.Type) {
		return errors.Errorf("check type '%s' is not supprted in V1", c.Type)
	}

	if err = validateQuery(c.Type, c.Query); err != nil {
		return err
	}

	if c.Family != "" {
		return errors.New("field 'family' is part of check format version 2 and can't be used in version 1")
	}

	if len(c.Queries) != 0 {
		return errors.New("field 'queries' is part of check format version 2 and can't be used in version 1")
	}

	return nil
}

func (c *Check) validateV2() error {
	var err error
	if err = c.Family.Validate(); err != nil {
		return err
	}

	if err = c.validateQueries(); err != nil {
		return err
	}

	if c.Type != "" {
		return errors.New("field 'type' is part of check format version 1 and can't be used in version 2")
	}

	if c.Query != "" {
		return errors.New("field 'query' is part of check format version 1 and can't be used in version 2")
	}

	return nil
}

func (c *Check) validateScript() error {
	if strings.ContainsRune(c.Script, '\t') {
		return errors.New("script should use spaces for indentation, not tabs")
	}

	return nil
}

func validateQuery(typ Type, query string) error {
	switch typ {
	case PostgreSQLShow, MongoDBGetParameter, MongoDBBuildInfo, MongoDBGetCmdLineOpts,
		MongoDBReplSetGetStatus, MongoDBGetDiagnosticData:
		if query != "" {
			return errors.Errorf("query should be empty for '%s' type", typ)
		}
	case PostgreSQLSelect, MySQLShow, MySQLSelect, ClickHouseSelect,
		MetricsInstant, MetricsRange:
		if query == "" {
			return errors.New("query is empty")
		}
	}

	return nil
}

func validateQueryParameters(typ Type, params map[Parameter]string) error {
	switch typ { //nolint:exhaustive
	case PostgreSQLShow, MongoDBGetParameter, MongoDBBuildInfo, MongoDBGetCmdLineOpts,
		MongoDBReplSetGetStatus, MongoDBGetDiagnosticData, MySQLShow, MySQLSelect:
		if len(params) != 0 {
			return errors.Errorf("query for '%s' type should not have any parameters", typ)
		}

	case PostgreSQLSelect:
		return validateParametersForPostgreSQLSelectQuery(params)

	case MetricsInstant:
		return validateParametersForMetricsInstantQuery(params)
	case MetricsRange:
		return validateParametersForMetricsRangeQuery(params)
	}

	return nil
}

func validateParametersForPostgreSQLSelectQuery(params map[Parameter]string) error {
	for param, value := range params {
		if param != AllDBs {
			return errors.Errorf("unsupported parameter '%s' for postgreSQL select query", param)
		}

		if _, err := strconv.ParseBool(value); err != nil {
			return errors.Wrapf(err, "failed to parse all_dbs parameter value %s, it should be a boolean", value)
		}
	}

	return nil
}

func validateParametersForMetricsInstantQuery(params map[Parameter]string) error {
	for param, value := range params {
		if param != Lookback {
			return errors.Errorf("unsupported parameter '%s' for instant metris query", param)
		}

		if _, err := time.ParseDuration(value); err != nil {
			return errors.Wrapf(err, "failed to parse loopback parameter value '%s', it should be a duration", value)
		}
	}

	return nil
}

func validateParametersForMetricsRangeQuery(params map[Parameter]string) error {
	if _, ok := params[Range]; !ok {
		return errors.New("query parameter 'range' is required for metrics range queries")
	}

	if _, ok := params[Step]; !ok {
		return errors.New("query parameter 'step' is required for metrics range queries")
	}

	for param, value := range params {
		switch param { //nolint:exhaustive
		case Lookback, Range, Step:
			if _, err := time.ParseDuration(value); err != nil {
				return errors.Wrapf(err, "failed to parse '%s' parameter value %s, it should be a duration", param, value)
			}
		default:
			return errors.Errorf("unsupported parameter '%s' for range metris query", param)
		}
	}

	return nil
}

func (c *Check) validateQueries() error {
	if len(c.Queries) == 0 {
		return errors.New("check should have at least one query")
	}

	var err error
	for _, q := range c.Queries {
		if err = q.Validate(); err != nil {
			return err
		}
	}

	switch c.Family {
	case MySQL:
		return checkQueryForCompatibilityWithMySQLFamily(c.Queries)
	case PostgreSQL:
		return checkQueryForCompatibilityWithPostgreSQLFamily(c.Queries)
	case MongoDB:
		return checkQueryCompatibilityWithMongoDBFamily(c.Queries)
	default:
		return errors.Errorf("unknown check family: %s", c.Family)
	}
}

func checkQueryForCompatibilityWithMySQLFamily(queries []Query) error {
	for _, q := range queries {
		switch q.Type { //nolint:exhaustive
		case MySQLShow:
		case MySQLSelect:
		case MetricsInstant:
		case MetricsRange:
		case ClickHouseSelect:
		default:
			return errors.Errorf("unsupported query type '%s' for mySQL family", q.Type)
		}
	}

	return nil
}

func checkQueryForCompatibilityWithPostgreSQLFamily(queries []Query) error {
	for _, q := range queries {
		switch q.Type { //nolint:exhaustive
		case PostgreSQLShow:
		case PostgreSQLSelect:
		case MetricsInstant:
		case MetricsRange:
		case ClickHouseSelect:
		default:
			return errors.Errorf("unsupported query type '%s' for postgreSQL family", q.Type)
		}
	}

	return nil
}

func checkQueryCompatibilityWithMongoDBFamily(queries []Query) error { //nolint:cyclop
	for _, q := range queries {
		switch q.Type { //nolint:exhaustive
		case MongoDBGetParameter:
		case MongoDBBuildInfo:
		case MongoDBGetCmdLineOpts:
		case MongoDBGetDiagnosticData:
		case MongoDBReplSetGetStatus:
		case MetricsInstant:
		case MetricsRange:
		case ClickHouseSelect:
		default:
			return errors.Errorf("unsupported query type '%s' for mongoDB family", q.Type)
		}
	}

	return nil
}
