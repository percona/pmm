package models

import (
	"database/sql"
	"encoding/json"

	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/sirupsen/logrus"
)

func encryptAgent(agent *Agent) {
	if agent.Username != nil {
		username, err := encryption.Encrypt(*agent.Username)
		if err != nil {
			logrus.Warning(err)
		}
		agent.Username = &username
	}

	if agent.Password != nil {
		password, err := encryption.Encrypt(*agent.Password)
		if err != nil {
			logrus.Warning(err)
		}
		agent.Password = &password
	}

	if agent.AgentPassword != nil {
		agentPassword, err := encryption.Encrypt(*agent.AgentPassword)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AgentPassword = &agentPassword
	}

	if agent.AWSAccessKey != nil {
		awsAccessKey, err := encryption.Encrypt(*agent.AWSAccessKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AWSAccessKey = &awsAccessKey
	}

	if agent.AWSSecretKey != nil {
		awsSecretKey, err := encryption.Encrypt(*agent.AWSSecretKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AWSSecretKey = &awsSecretKey
	}

	var err error
	if agent.MySQLOptions != nil {
		agent.MySQLOptions.TLSKey, err = encryption.Encrypt(agent.MySQLOptions.TLSKey)
		if err != nil {
			logrus.Warning(err)
		}

	}

	if agent.PostgreSQLOptions != nil {
		agent.PostgreSQLOptions.SSLKey, err = encryption.Encrypt(agent.PostgreSQLOptions.SSLKey)
		if err != nil {
			logrus.Warning(err)
		}
	}

	if agent.MongoDBOptions != nil {
		agent.MongoDBOptions.TLSCertificateKey, err = encryption.Encrypt(agent.MongoDBOptions.TLSCertificateKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.MongoDBOptions.TLSCertificateKeyFilePassword, err = encryption.Encrypt(agent.MongoDBOptions.TLSCertificateKeyFilePassword)
		if err != nil {
			logrus.Warning(err)
		}
	}

	if agent.AzureOptions != nil {
		agent.AzureOptions.ClientID, err = encryption.Encrypt(agent.AzureOptions.ClientID)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.ClientSecret, err = encryption.Encrypt(agent.AzureOptions.ClientSecret)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.SubscriptionID, err = encryption.Encrypt(agent.AzureOptions.SubscriptionID)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.TenantID, err = encryption.Encrypt(agent.AzureOptions.TenantID)
		if err != nil {
			logrus.Warning(err)
		}
	}
}

func decryptAgent(agent *Agent) {
	if agent.Username != nil {
		username, err := encryption.Decrypt(*agent.Username)
		if err != nil {
			logrus.Warning(err)
		}
		agent.Username = &username
	}

	if agent.Password != nil {
		password, err := encryption.Decrypt(*agent.Password)
		if err != nil {
			logrus.Warning(err)
		}
		agent.Password = &password
	}

	if agent.AgentPassword != nil {
		agentPassword, err := encryption.Decrypt(*agent.AgentPassword)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AgentPassword = &agentPassword
	}

	if agent.AWSAccessKey != nil {
		awsAccessKey, err := encryption.Decrypt(*agent.AWSAccessKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AWSAccessKey = &awsAccessKey
	}

	if agent.AWSSecretKey != nil {
		awsSecretKey, err := encryption.Decrypt(*agent.AWSSecretKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AWSSecretKey = &awsSecretKey
	}

	var err error
	if agent.MySQLOptions != nil {
		agent.MySQLOptions.TLSKey, err = encryption.Decrypt(agent.MySQLOptions.TLSKey)
		if err != nil {
			logrus.Warning(err)
		}

	}

	if agent.PostgreSQLOptions != nil {
		agent.PostgreSQLOptions.SSLKey, err = encryption.Decrypt(agent.PostgreSQLOptions.SSLKey)
		if err != nil {
			logrus.Warning(err)
		}
	}

	if agent.MongoDBOptions != nil {
		agent.MongoDBOptions.TLSCertificateKey, err = encryption.Decrypt(agent.MongoDBOptions.TLSCertificateKey)
		if err != nil {
			logrus.Warning(err)
		}
		agent.MongoDBOptions.TLSCertificateKeyFilePassword, err = encryption.Decrypt(agent.MongoDBOptions.TLSCertificateKeyFilePassword)
		if err != nil {
			logrus.Warning(err)
		}
	}

	if agent.AzureOptions != nil {
		agent.AzureOptions.ClientID, err = encryption.Decrypt(agent.AzureOptions.ClientID)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.ClientSecret, err = encryption.Decrypt(agent.AzureOptions.ClientSecret)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.SubscriptionID, err = encryption.Decrypt(agent.AzureOptions.SubscriptionID)
		if err != nil {
			logrus.Warning(err)
		}
		agent.AzureOptions.TenantID, err = encryption.Decrypt(agent.AzureOptions.TenantID)
		if err != nil {
			logrus.Warning(err)
		}
	}
}

func EncryptColumnPostgreSQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	o := PostgreSQLOptions{}
	value := val.(*sql.NullString)
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.SSLCa, err = e.Encrypt(o.SSLCa)
	if err != nil {
		return nil, err
	}
	o.SSLCert, err = e.Encrypt(o.SSLCert)
	if err != nil {
		return nil, err
	}
	o.SSLKey, err = encryption.Encrypt(o.SSLKey)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}
