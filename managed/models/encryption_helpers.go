package models

import (
	"database/sql"
	"encoding/json"

	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/sirupsen/logrus"
)

func encryptAgent(agent *Agent) {
	agentEncryption(agent, encryption.Encrypt)
}

func decryptAgent(agent *Agent) {
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

func EncryptMySQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mySQLOptionsHandler(val, e.Encrypt)
}

func DecryptMySQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mySQLOptionsHandler(val, e.Decrypt)
}

func mySQLOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := MySQLOptions{}
	value := val.(*sql.NullString)
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

func EncryptPostgreSQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return postgreSQLOptionsHandler(val, e.Encrypt)
}

func DecryptPostgreSQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return postgreSQLOptionsHandler(val, e.Decrypt)
}

func postgreSQLOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := PostgreSQLOptions{}
	value := val.(*sql.NullString)
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

func EncryptMongoDBOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mongoDBOptionsHandler(val, e.Encrypt)
}

func DecryptMongoDBOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mongoDBOptionsHandler(val, e.Decrypt)
}

func mongoDBOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := MongoDBOptions{}
	value := val.(*sql.NullString)
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

func EncryptAzureOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return azureOptionsHandler(val, e.Encrypt)
}

func DecryptAzureOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return azureOptionsHandler(val, e.Decrypt)
}

func azureOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := AzureOptions{}
	value := val.(*sql.NullString)
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