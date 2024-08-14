// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package actions

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestNewMongoExplain(t *testing.T) {
	database := "test"
	collection := "test_col"
	id := "abcd1234"
	ctx := context.TODO()

	dsn := tests.GetTestMongoDBDSN(t)
	client := tests.OpenTestMongoDB(t, dsn)
	t.Cleanup(func() { defer client.Disconnect(ctx) }) //nolint:errcheck
	defer client.Database(database).Drop(ctx)          //nolint:errcheck

	err := prepareData(ctx, client, database, collection)
	require.NoError(t, err)

	params := &agentpb.StartActionRequest_MongoDBExplainParams{
		Dsn: tests.GetTestMongoDBDSN(t),
		Query: `
		{
			"ns": "testdb.listingsAndReviews",
			"op": "command",
			"command": {
			  "explain": {
				"find": "listingsAndReviews",
				"filter": {
				  "$and": [
					{
					  "repositoryFilePath": {
						"$ne": ""
					  }
					},
					{
					  "repositoryFilePath": {
						"$ne": null
					  },
					  "delete": {
						"$ne": true
					  }
					},
					{
					  "$and": [
						{
						  "num_abonado": "985662747"
						},
						{
						  "fecha_emision": {
							"$gte": {
							  "$date": {
								"$numberLong": "1695160800000"
							  }
							}
						  }
						},
						{
						  "fecha_emision": {
							"$lte": {
							  "$date": {
								"$numberLong": "1697839199000"
							  }
							}
						  }
						},
						{
						  "nif": {
							"$in": [
							  "B74145558",
							  "LB74145558",
							  "L00B74145558",
							  "B74145558",
							  "LB74145558",
							  "L00B74145558"
							]
						  },
						  "shardingKey": {
							"$in": [
							  "B74145558",
							  "LB74145558",
							  "L00B74145558",
							  "B74145558",
							  "LB74145558",
							  "L00B74145558"
							]
						  },
						  "prefix_shardingKey": {
							"$in": [
							  "5621",
							  "d266",
							  "2ced",
							  "5621",
							  "d266",
							  "2ced"
							]
						  }
						},
						{
						  "origen_documento": "Facturacion"
						},
						{
						  "tipo_documento": {
							"$in": [
							  "FACTURA-REGULAR",
							  "FACTURA-RF",
							  "FACTURA-RF-ANULADA",
							  "RESUMEN-CONC-FIJO",
							  "RESUMEN-CONC-REGULAR",
							  "RESUMEN-CONC-VAR",
							  "RESUMEN-CTOS-INDIV",
							  "RESUMEN-DIRECTA-TDE",
							  "RESUMEN-FTNR-A-CONC",
							  "RESUMEN-FTNR-A-CTOS",
							  "RESUMEN-FTNR-A-I2",
							  "RESUMEN-FTNR-A-PERS",
							  "RESUMEN-FTNR-A-RI",
							  "RESUMEN-FTNR-A-STB",
							  "RESUMEN-FTNR-CONC",
							  "RESUMEN-FTNR-CTOS",
							  "RESUMEN-FTNR-IBERCOM",
							  "RESUMEN-FTNR-PERS",
							  "RESUMEN-FTNR-RI",
							  "RESUMEN-FTNR-RPV",
							  "RESUMEN-FTNR-STB",
							  "RESUMEN-MAR-NACIONAL",
							  "RESUMEN-PERS",
							  "RESUMEN-REF-CLIENTE-CTOS",
							  "RESUMEN-REF-CLIENTE-TD",
							  "RESUMEN-VE-ANULACION",
							  "RESUMEN-VENTA-EQUIPOS",
							  "TS-ANULACION",
							  "TS-ANULACION-OTRAS",
							  "TS-ANULACION-OTRAS-SD",
							  "TS-ANULACION-SD",
							  "TS-FACTURAS-EMITIDAS",
							  "TS-FACTURAS-EMITIDAS-OTRAS",
							  "TS-FACTURAS-EMITIDAS-OTRAS-SD",
							  "TS-FACTURAS-EMITIDAS-SD",
							  "TS-FACTURAS-INHIB",
							  "TS-FACTURAS-INHIB-OTRAS",
							  "TS-FACTURAS-INHIB-OTRAS-SD",
							  "TS-FACTURAS-INHIB-SD",
							  "FACTURA-TSOL",
							  "FACTURA"
							]
						  }
						}
					  ]
					}
				  ]
				},
				"lsid": {
				  "id": {
					"$binary": {
					  "base64": "+znG58SsRj2RcxF+d+jFsA==",
					  "subType": "04"
					}
				  }
				},
				"$db": "testdb"
			  },
			  "lsid": {
				"id": {
				  "$binary": {
					"base64": "3dAPy58ITpmFXX2/fhKP0w==",
					"subType": "04"
				  }
				}
			  },
			  "$db": "testdb"
			}
		  }`,
	}

	ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
	require.NoError(t, err)

	res, err := ex.Run(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, string(res))
}
func prepareData(ctx context.Context, client *mongo.Client, database, collection string) error {
	max := int64(100)
	count, _ := client.Database(database).Collection(collection).CountDocuments(ctx, nil)

	if count < max {
		for i := int64(0); i < max; i++ {
			doc := primitive.M{"f1": i, "f2": fmt.Sprintf("text_%5d", max-i)}
			if _, err := client.Database(database).Collection(collection).InsertOne(ctx, doc); err != nil {
				return err
			}
		}
	}

	return nil
}
