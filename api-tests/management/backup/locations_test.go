// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package backup

import (
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	backupClient "github.com/percona/pmm/api/managementpb/backup/json/client"
	"github.com/percona/pmm/api/managementpb/backup/json/client/locations"
)

func TestAddLocation(t *testing.T) {
	t.Parallel()
	client := backupClient.Default.Locations

	t.Run("normal pmm client config", func(t *testing.T) {
		t.Parallel()

		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:        gofakeit.Name(),
				Description: gofakeit.Question(),
				PMMClientConfig: &locations.AddLocationParamsBodyPMMClientConfig{
					Path: "/tmp",
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteLocation(t, client, resp.Payload.LocationID)

		assert.NotEmpty(t, resp.Payload.LocationID)
	})

	t.Run("normal pmm server config", func(t *testing.T) {
		t.Parallel()

		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:        gofakeit.Name(),
				Description: gofakeit.Question(),
				PMMServerConfig: &locations.AddLocationParamsBodyPMMServerConfig{
					Path: "/tmp",
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteLocation(t, client, resp.Payload.LocationID)

		assert.NotEmpty(t, resp.Payload.LocationID)
	})

	t.Run("normal s3 config", func(t *testing.T) {
		t.Parallel()
		accessKey, secretKey, bucketName := os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_SECRET_KEY"), os.Getenv("AWS_BUCKET_NAME")
		if accessKey == "" || secretKey == "" || bucketName == "" {
			t.Skip("Skipping add S3 backup location - missing credentials")
		}
		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:        gofakeit.Name(),
				Description: gofakeit.Question(),
				S3Config: &locations.AddLocationParamsBodyS3Config{
					Endpoint:   "https://s3.us-west-2.amazonaws.com",
					AccessKey:  accessKey,
					SecretKey:  secretKey,
					BucketName: bucketName,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteLocation(t, client, resp.Payload.LocationID)

		assert.NotEmpty(t, resp.Payload.LocationID)
	})
}

func TestAddWrongLocation(t *testing.T) {
	t.Parallel()
	client := backupClient.Default.Locations

	t.Run("missing config", func(t *testing.T) {
		t.Parallel()

		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:        gofakeit.Name(),
				Description: gofakeit.Question(),
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Missing location config.")
		assert.Nil(t, resp)
	})

	t.Run("missing client config path", func(t *testing.T) {
		t.Parallel()

		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:            gofakeit.Name(),
				Description:     gofakeit.Question(),
				PMMClientConfig: &locations.AddLocationParamsBodyPMMClientConfig{},
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field PmmClientConfig.Path: value '' must not be an empty string")
		assert.Nil(t, resp)
	})

	t.Run("missing name", func(t *testing.T) {
		t.Parallel()

		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Description: gofakeit.Question(),
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field Name: value '' must not be an empty string")
		assert.Nil(t, resp)
	})

	t.Run("missing s3 endpoint", func(t *testing.T) {
		t.Parallel()

		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:        gofakeit.Name(),
				Description: gofakeit.Question(),
				S3Config: &locations.AddLocationParamsBodyS3Config{
					AccessKey:  "access_key",
					SecretKey:  "secret_key",
					BucketName: "example_bucket",
				},
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field S3Config.Endpoint: value '' must not be an empty string")
		assert.Nil(t, resp)
	})

	t.Run("missing s3 bucket", func(t *testing.T) {
		t.Parallel()

		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:        gofakeit.Name(),
				Description: gofakeit.Question(),
				S3Config: &locations.AddLocationParamsBodyS3Config{
					Endpoint:  "http://example.com",
					AccessKey: "access_key",
					SecretKey: "secret_key",
				},
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field S3Config.BucketName: value '' must not be an empty string")
		assert.Nil(t, resp)
	})

	t.Run("double config", func(t *testing.T) {
		t.Parallel()

		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:        gofakeit.Name(),
				Description: gofakeit.Question(),
				PMMClientConfig: &locations.AddLocationParamsBodyPMMClientConfig{
					Path: "/tmp",
				},
				S3Config: &locations.AddLocationParamsBodyS3Config{
					Endpoint:   "http://example.com",
					AccessKey:  "access_key",
					SecretKey:  "secret_key",
					BucketName: "example_bucket",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Only one config is allowed.")

		assert.Nil(t, resp)
	})
}

func TestListLocations(t *testing.T) {
	t.Parallel()
	client := backupClient.Default.Locations

	body := locations.AddLocationBody{
		Name:        gofakeit.Name(),
		Description: gofakeit.Question(),
		PMMClientConfig: &locations.AddLocationParamsBodyPMMClientConfig{
			Path: "/tmp",
		},
	}
	addResp, err := client.AddLocation(&locations.AddLocationParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	defer deleteLocation(t, client, addResp.Payload.LocationID)

	resp, err := client.ListLocations(&locations.ListLocationsParams{Context: pmmapitests.Context})
	require.NoError(t, err)

	assert.NotEmpty(t, resp.Payload.Locations)
	var found bool
	for _, loc := range resp.Payload.Locations {
		if loc.LocationID == addResp.Payload.LocationID {
			assert.Equal(t, body.Name, loc.Name)
			assert.Equal(t, body.Description, loc.Description)
			assert.Equal(t, body.PMMClientConfig.Path, loc.PMMClientConfig.Path)
			found = true
		}
	}
	assert.True(t, found, "Expected location not found")
}

func TestChangeLocation(t *testing.T) {
	t.Parallel()
	client := backupClient.Default.Locations

	checkChange := func(t *testing.T, req locations.ChangeLocationBody, locations []*locations.ListLocationsOKBodyLocationsItems0) {
		t.Helper()
		var found bool
		for _, loc := range locations {
			if loc.LocationID == req.LocationID {
				assert.Equal(t, req.Name, loc.Name)
				if req.Description != "" {
					assert.Equal(t, req.Description, loc.Description)
				}

				if req.PMMServerConfig != nil {
					require.NotNil(t, loc.PMMServerConfig)
					assert.Equal(t, req.PMMServerConfig.Path, loc.PMMServerConfig.Path)
				} else {
					assert.Nil(t, loc.PMMServerConfig)
				}

				if req.PMMClientConfig != nil {
					require.NotNil(t, loc.PMMClientConfig)
					assert.Equal(t, req.PMMClientConfig.Path, loc.PMMClientConfig.Path)
				} else {
					assert.Nil(t, loc.PMMClientConfig)
				}

				if req.S3Config != nil {
					require.NotNil(t, loc.S3Config)
					assert.Equal(t, req.S3Config.Endpoint, loc.S3Config.Endpoint)
					assert.Equal(t, req.S3Config.AccessKey, loc.S3Config.AccessKey)
					assert.Equal(t, req.S3Config.SecretKey, loc.S3Config.SecretKey)
					assert.Equal(t, req.S3Config.BucketName, loc.S3Config.BucketName)
				} else {
					assert.Nil(t, loc.S3Config)
				}

				found = true

				break
			}
		}
		assert.True(t, found)
	}
	t.Run("update name and config path", func(t *testing.T) {
		t.Parallel()

		addReqBody := locations.AddLocationBody{
			Name:        gofakeit.Name(),
			Description: gofakeit.Question(),
			PMMServerConfig: &locations.AddLocationParamsBodyPMMServerConfig{
				Path: "/tmp",
			},
		}
		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body:    addReqBody,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteLocation(t, client, resp.Payload.LocationID)

		updateBody := locations.ChangeLocationBody{
			LocationID: resp.Payload.LocationID,
			Name:       gofakeit.Name(),
			PMMServerConfig: &locations.ChangeLocationParamsBodyPMMServerConfig{
				Path: "/tmp/nested",
			},
		}
		_, err = client.ChangeLocation(&locations.ChangeLocationParams{
			Body:    updateBody,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		listResp, err := client.ListLocations(&locations.ListLocationsParams{Context: pmmapitests.Context})
		require.NoError(t, err)

		checkChange(t, updateBody, listResp.Payload.Locations)
	})

	t.Run("update only name", func(t *testing.T) {
		t.Parallel()

		addReqBody := locations.AddLocationBody{
			Name:        gofakeit.Name(),
			Description: gofakeit.Question(),
			PMMServerConfig: &locations.AddLocationParamsBodyPMMServerConfig{
				Path: "/tmp",
			},
		}
		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body:    addReqBody,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteLocation(t, client, resp.Payload.LocationID)

		updateBody := locations.ChangeLocationBody{
			LocationID: resp.Payload.LocationID,
			Name:       gofakeit.Name(),
		}
		_, err = client.ChangeLocation(&locations.ChangeLocationParams{
			Body:    updateBody,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		listResp, err := client.ListLocations(&locations.ListLocationsParams{Context: pmmapitests.Context})
		require.NoError(t, err)

		var location *locations.ListLocationsOKBodyLocationsItems0
		for _, loc := range listResp.Payload.Locations {
			if loc.LocationID == resp.Payload.LocationID {
				location = loc
				break
			}
		}
		require.NotNil(t, location)

		assert.Equal(t, location.Name, updateBody.Name)
		require.NotNil(t, location.PMMServerConfig)
		assert.Equal(t, addReqBody.PMMServerConfig.Path, location.PMMServerConfig.Path)
	})

	t.Run("change config type", func(t *testing.T) {
		t.Parallel()

		addReqBody := locations.AddLocationBody{
			Name:        gofakeit.Name(),
			Description: gofakeit.Question(),
			PMMServerConfig: &locations.AddLocationParamsBodyPMMServerConfig{
				Path: "/tmp",
			},
		}
		resp, err := client.AddLocation(&locations.AddLocationParams{
			Body:    addReqBody,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteLocation(t, client, resp.Payload.LocationID)

		updateBody := locations.ChangeLocationBody{
			LocationID: resp.Payload.LocationID,
			Name:       gofakeit.Name(),
			PMMClientConfig: &locations.ChangeLocationParamsBodyPMMClientConfig{
				Path: "/root",
			},
		}
		_, err = client.ChangeLocation(&locations.ChangeLocationParams{
			Body:    updateBody,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		listResp, err := client.ListLocations(&locations.ListLocationsParams{Context: pmmapitests.Context})
		require.NoError(t, err)

		checkChange(t, updateBody, listResp.Payload.Locations)
	})

	t.Run("change to existing name - error", func(t *testing.T) {
		t.Parallel()

		addReqBody1 := locations.AddLocationBody{
			Name:        gofakeit.Name(),
			Description: gofakeit.Question(),
			PMMServerConfig: &locations.AddLocationParamsBodyPMMServerConfig{
				Path: "/tmp",
			},
		}
		resp1, err := client.AddLocation(&locations.AddLocationParams{
			Body:    addReqBody1,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteLocation(t, client, resp1.Payload.LocationID)

		addReqBody2 := locations.AddLocationBody{
			Name:        gofakeit.Name(),
			Description: gofakeit.Question(),
			PMMServerConfig: &locations.AddLocationParamsBodyPMMServerConfig{
				Path: "/tmp",
			},
		}
		resp2, err := client.AddLocation(&locations.AddLocationParams{
			Body:    addReqBody2,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteLocation(t, client, resp2.Payload.LocationID)

		updateBody := locations.ChangeLocationBody{
			LocationID: resp2.Payload.LocationID,
			Name:       addReqBody1.Name,
			PMMServerConfig: &locations.ChangeLocationParamsBodyPMMServerConfig{
				Path: "/tmp",
			},
		}
		_, err = client.ChangeLocation(&locations.ChangeLocationParams{
			Body:    updateBody,
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, `Location with name "%s" already exists.`, updateBody.Name)
	})
}

func TestRemoveLocation(t *testing.T) {
	t.Parallel()
	client := backupClient.Default.Locations
	resp, err := client.AddLocation(&locations.AddLocationParams{
		Body: locations.AddLocationBody{
			Name:        gofakeit.Name(),
			Description: gofakeit.Question(),
			PMMClientConfig: &locations.AddLocationParamsBodyPMMClientConfig{
				Path: "/tmp",
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)

	_, err = client.RemoveLocation(&locations.RemoveLocationParams{
		Body: locations.RemoveLocationBody{
			LocationID: resp.Payload.LocationID,
			Force:      false,
		},
		Context: pmmapitests.Context,
	})

	require.NoError(t, err)

	assertNotFound := func(id string, locations []*locations.ListLocationsOKBodyLocationsItems0) func() bool {
		return func() bool {
			for _, loc := range locations {
				if loc.LocationID == id {
					return false
				}
			}
			return true
		}
	}

	listResp, err := client.ListLocations(&locations.ListLocationsParams{Context: pmmapitests.Context})
	require.NoError(t, err)

	assert.Condition(t, assertNotFound(resp.Payload.LocationID, listResp.Payload.Locations))
}

func TestLocationConfigValidation(t *testing.T) {
	t.Parallel()
	client := backupClient.Default.Locations

	t.Run("missing config", func(t *testing.T) {
		t.Parallel()

		resp, err := client.TestLocationConfig(&locations.TestLocationConfigParams{
			Body:    locations.TestLocationConfigBody{},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Missing location config.")
		assert.Nil(t, resp)
	})

	t.Run("missing client config path", func(t *testing.T) {
		t.Parallel()

		resp, err := client.TestLocationConfig(&locations.TestLocationConfigParams{
			Body: locations.TestLocationConfigBody{
				PMMClientConfig: &locations.TestLocationConfigParamsBodyPMMClientConfig{},
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field PmmClientConfig.Path: value '' must not be an empty string")
		assert.Nil(t, resp)
	})

	t.Run("missing s3 endpoint", func(t *testing.T) {
		t.Parallel()

		resp, err := client.TestLocationConfig(&locations.TestLocationConfigParams{
			Body: locations.TestLocationConfigBody{
				S3Config: &locations.TestLocationConfigParamsBodyS3Config{
					AccessKey:  "access_key",
					SecretKey:  "secret_key",
					BucketName: "example_bucket",
				},
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field S3Config.Endpoint: value '' must not be an empty string")
		assert.Nil(t, resp)
	})

	t.Run("missing s3 bucket", func(t *testing.T) {
		t.Parallel()

		resp, err := client.TestLocationConfig(&locations.TestLocationConfigParams{
			Body: locations.TestLocationConfigBody{
				S3Config: &locations.TestLocationConfigParamsBodyS3Config{
					Endpoint:  "http://example.com",
					AccessKey: "access_key",
					SecretKey: "secret_key",
				},
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field S3Config.BucketName: value '' must not be an empty string")
		assert.Nil(t, resp)
	})

	t.Run("double config", func(t *testing.T) {
		t.Parallel()

		resp, err := client.TestLocationConfig(&locations.TestLocationConfigParams{
			Body: locations.TestLocationConfigBody{
				PMMClientConfig: &locations.TestLocationConfigParamsBodyPMMClientConfig{
					Path: "/tmp",
				},
				S3Config: &locations.TestLocationConfigParamsBodyS3Config{
					Endpoint:   "http://example.com",
					AccessKey:  "access_key",
					SecretKey:  "secret_key",
					BucketName: "example_bucket",
				},
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Only one config is allowed.")

		assert.Nil(t, resp)
	})
}

func deleteLocation(t *testing.T, client locations.ClientService, id string) {
	t.Helper()
	_, err := client.RemoveLocation(&locations.RemoveLocationParams{
		Body: locations.RemoveLocationBody{
			LocationID: id,
			Force:      false,
		},
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
}
