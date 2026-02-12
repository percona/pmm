package realtimeanalytics

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	// Timeout for establishing connection to MongoDB.
	mgoConnectTimeout = 5 * time.Second
	// Timeout for MongoDB queries.
	mgoQueryTimeout = 5 * time.Second
)

// createSession creates new MongoDB client and checks connection to MongoDB by pinging it.
func createSession(ctx context.Context, dsn string, agentID string) (*mongo.Client, error) {
	opts, err := clientOptionsForDSN(dsn)
	if err != nil {
		return nil, err
	}

	opts = opts.
		SetDirect(true).
		SetReadPreference(readpref.Nearest()).
		SetTimeout(mgoQueryTimeout).
		SetConnectTimeout(mgoConnectTimeout).
		SetCompressors([]string{"snappy", "zlib", "zstd"}).
		SetAppName(fmt.Sprintf("RTA-mongodb-%s", agentID))

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}

	if err = client.Ping(ctx, readpref.Nearest()); err != nil {
		return nil, err
	}

	return client, nil
}

// ClientOptionsForDSN applies URI to Client.
func clientOptionsForDSN(dsn string) (*options.ClientOptions, error) {
	clientOptions := options.Client().ApplyURI(dsn)
	if e := clientOptions.Validate(); e != nil {
		return nil, e
	}

	// Workaround for PMM-9320
	// if username or password is set, need to replace it with correctly parsed credentials.
	parsedDsn, err := url.Parse(dsn)
	if err != nil {
		// for non-URI, do nothing (PMM-10265)
		return clientOptions, nil //nolint:nilerr
	}
	username := parsedDsn.User.Username()
	password, _ := parsedDsn.User.Password()
	if username != "" || password != "" {
		clientOptions.Auth.Username = username
		clientOptions.Auth.Password = password
	}

	return clientOptions, nil
}
