package database

const (
	// SetupFixtures adds initial data to the database.
	SetupFixtures SetupFixturesMode = iota
	// SkipFixtures skips adding initial data to the database. Useful for tests.
	SkipFixtures

	// DisableSSLMode represent disable PostgreSQL ssl mode.
	DisableSSLMode string = "disable"
	// RequireSSLMode represent require PostgreSQL ssl mode.
	RequireSSLMode string = "require"
	// VerifyCaSSLMode represent verify-ca PostgreSQL ssl mode.
	VerifyCaSSLMode string = "verify-ca"
	// VerifyFullSSLMode represent verify-full PostgreSQL ssl mode.
	VerifyFullSSLMode string = "verify-full"
)

// SetupFixturesMode defines if SetupDB adds initial data to the database or not.
type SetupFixturesMode int
