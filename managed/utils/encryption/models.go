package encryption

import "github.com/google/tink/go/tink"

type Encryption struct {
	Path      string
	Key       string
	Primitive tink.AEAD
}

type DatabaseConnection struct {
	Host, User, Password string
	Port                 int16
	DBName               string
	SSLMode              string
	SSLCAPath            string
	SSLKeyPath           string
	SSLCertPath          string
	EncryptedItems       []EncryptedItem
}

type EncryptedItem struct {
	Database, Table string
	Identificators  []string
	Columns         []string
}

type QueryValues struct {
	Query       string
	SetValues   [][]any
	WhereValues [][]any
}
