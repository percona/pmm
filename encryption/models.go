package encryption

type Encryption struct {
	Path string
	Key  string
}

type DatabaseConnection struct {
	Host, User, Password string
	Port                 int16
	SSLMode              string
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
