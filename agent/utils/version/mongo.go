package version

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/version"
)

// GetMongoDBVersion returns the parsed version of the connected MongoDB server.
func GetMongoDBVersion(ctx context.Context, client *mongo.Client) (*version.Parsed, error) {
	resp := client.Database("admin").RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}})
	if err := resp.Err(); err != nil {
		return nil, err
	}

	buildInfo := struct {
		Version string `bson:"version"`
	}{}

	if err := resp.Decode(&buildInfo); err != nil {
		return nil, err
	}

	mongoVersion, err := version.Parse(buildInfo.Version)
	if err != nil {
		return nil, err
	}
	return mongoVersion, nil
}
