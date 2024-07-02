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

package models

import (
	"database/sql"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/encryption"
)

// EncryptAgent encrypt agent.
func EncryptAgent(agent *Agent) {
	agentEncryption(agent, encryption.Encrypt)
}

// DecryptAgent decrypt agent.
func DecryptAgent(agent *Agent) {
	agentEncryption(agent, encryption.Decrypt)
}

func agentEncryption(agent *Agent, handler func(string) (string, error)) {
	if agent.Username != nil {
		username, err := handler(*agent.Username)
		if err != nil {
			logrus.Warning(err)
		}
		agent.Username = &username
	}

	if agent.Password != nil {
		password, err := handler(*agent.Password)
		if err != nil {
			logrus.Warning(err)
		}
		agent.Password = &password
	}

	if agent.AgentPassword != nil {
		agentPassword, err := handler(*agent.AgentPassword)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AgentPassword = &agentPassword
	}

	if agent.AWSAccessKey != nil {
		awsAccessKey, err := handler(*agent.AWSAccessKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AWSAccessKey = &awsAccessKey
	}

	if agent.AWSSecretKey != nil {
		awsSecretKey, err := handler(*agent.AWSSecretKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AWSSecretKey = &awsSecretKey
	}

	var err error
	if agent.MySQLOptions != nil {
		agent.MySQLOptions.TLSCa, err = handler(agent.MySQLOptions.TLSCa)
		if err != nil {
			logrus.Warning(err)
		}
		agent.MySQLOptions.TLSCert, err = handler(agent.MySQLOptions.TLSCert)
		if err != nil {
			logrus.Warning(err)
		}
		agent.MySQLOptions.TLSKey, err = handler(agent.MySQLOptions.TLSKey)
		if err != nil {
			logrus.Warning(err)
		}
	}

	if agent.PostgreSQLOptions != nil {
		agent.PostgreSQLOptions.SSLCa, err = handler(agent.PostgreSQLOptions.SSLCa)
		if err != nil {
			logrus.Warning(err)
		}
		agent.PostgreSQLOptions.SSLCert, err = handler(agent.PostgreSQLOptions.SSLCert)
		if err != nil {
			logrus.Warning(err)
		}
		agent.PostgreSQLOptions.SSLKey, err = handler(agent.PostgreSQLOptions.SSLKey)
		if err != nil {
			logrus.Warning(err)
		}
	}

	if agent.MongoDBOptions != nil {
		agent.MongoDBOptions.TLSCa, err = handler(agent.MongoDBOptions.TLSCa)
		if err != nil {
			logrus.Warning(err)
		}
		agent.MongoDBOptions.TLSCertificateKey, err = handler(agent.MongoDBOptions.TLSCertificateKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.MongoDBOptions.TLSCertificateKeyFilePassword, err = handler(agent.MongoDBOptions.TLSCertificateKeyFilePassword)
		if err != nil {
			logrus.Warning(err)
		}
	}

	if agent.AzureOptions != nil {
		agent.AzureOptions.ClientID, err = handler(agent.AzureOptions.ClientID)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.ClientSecret, err = handler(agent.AzureOptions.ClientSecret)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.SubscriptionID, err = handler(agent.AzureOptions.SubscriptionID)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.TenantID, err = handler(agent.AzureOptions.TenantID)
		if err != nil {
			logrus.Warning(err)
		}
	}
}

// EncryptMySQLOptionsHandler returns encrypted MySQL Options.
func EncryptMySQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mySQLOptionsHandler(val, e.Encrypt)
}

// DecryptMySQLOptionsHandler returns decrypted MySQL Options.
func DecryptMySQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mySQLOptionsHandler(val, e.Decrypt)
}

func mySQLOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := MySQLOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.TLSCa, err = handler(o.TLSCa)
	if err != nil {
		return nil, err
	}
	o.TLSCert, err = handler(o.TLSCert)
	if err != nil {
		return nil, err
	}
	o.TLSKey, err = handler(o.TLSKey)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EncryptPostgreSQLOptionsHandler returns encrypted PostgreSQL Options.
func EncryptPostgreSQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return postgreSQLOptionsHandler(val, e.Encrypt)
}

// DecryptPostgreSQLOptionsHandler returns decrypted PostgreSQL Options.
func DecryptPostgreSQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return postgreSQLOptionsHandler(val, e.Decrypt)
}

func postgreSQLOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := PostgreSQLOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.SSLCa, err = handler(o.SSLCa)
	if err != nil {
		return nil, err
	}
	o.SSLCert, err = handler(o.SSLCert)
	if err != nil {
		return nil, err
	}
	o.SSLKey, err = handler(o.SSLKey)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EncryptMongoDBOptionsHandler returns encrypted MongoDB Options.
func EncryptMongoDBOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mongoDBOptionsHandler(val, e.Encrypt)
}

// DecryptMongoDBOptionsHandler returns decrypted MongoDB Options.
func DecryptMongoDBOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mongoDBOptionsHandler(val, e.Decrypt)
}

func mongoDBOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := MongoDBOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.TLSCa, err = handler(o.TLSCa)
	if err != nil {
		return nil, err
	}
	o.TLSCertificateKey, err = handler(o.TLSCertificateKey)
	if err != nil {
		return nil, err
	}
	o.TLSCertificateKeyFilePassword, err = handler(o.TLSCertificateKeyFilePassword)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EncryptAzureOptionsHandler returns encrypted Azure Options.
func EncryptAzureOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return azureOptionsHandler(val, e.Encrypt)
}

// DecryptAzureOptionsHandler returns decrypted Azure Options.
func DecryptAzureOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return azureOptionsHandler(val, e.Decrypt)
}

func azureOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := AzureOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.ClientID, err = handler(o.ClientID)
	if err != nil {
		return nil, err
	}
	o.ClientSecret, err = handler(o.ClientSecret)
	if err != nil {
		return nil, err
	}
	o.SubscriptionID, err = handler(o.SubscriptionID)
	if err != nil {
		return nil, err
	}
	o.TenantID, err = handler(o.TenantID)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}
