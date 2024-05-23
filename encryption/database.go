package encryption

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DatabaseConnection struct {
	Host, User, Password string
	Port                 int16
	EncryptedItems       []EncryptedItem
}

type EncryptedItem struct {
	Database, Table string
	Columns         []string
}

func (c DatabaseConnection) Connect() error {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable", c.Host, c.Port, c.User, c.Password)
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}

	return nil
}
