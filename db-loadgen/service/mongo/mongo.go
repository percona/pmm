package mongo

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProgressTracker interface {
	AddMongoOp()
	AddMongoError()
}

type Service struct {
	dsn     string
	workers int
	tracker ProgressTracker
}

func New(dsn string, workers int, tracker ProgressTracker) *Service {
	return &Service{
		dsn:     dsn,
		workers: workers,
		tracker: tracker,
	}
}

func (s *Service) StartLoad(ctx context.Context, wg *sync.WaitGroup) {
	fmt.Printf("Starting MongoDB load with %d workers\n", s.workers)

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go s.worker(ctx, wg, i)
	}
}

func (s *Service) worker(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(s.dsn))
	if err != nil {
		log.Printf("MongoDB worker %d: failed to connect: %v", id, err)
		return
	}
	defer client.Disconnect(ctx)

	operations := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := s.performUnoptimizedOperation(ctx, client); err != nil {
				// Only log unexpected errors, not intentional anti-pattern errors
				if s.isExpectedAntiPatternError(err) {
					operations++ // Count as successful anti-pattern demonstration
					if s.tracker != nil {
						s.tracker.AddMongoOp()
					}
				} else {
					log.Printf("MongoDB worker %d: unexpected error: %v", id, err)
					if s.tracker != nil {
						s.tracker.AddMongoError()
					}
				}
			} else {
				operations++
				if s.tracker != nil {
					s.tracker.AddMongoOp()
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// isExpectedAntiPatternError checks if the error is an expected result of our anti-patterns
func (s *Service) isExpectedAntiPatternError(err error) bool {
	errStr := err.Error()

	// Common anti-pattern errors we expect to see
	expectedErrors := []string{
		"multi-key map passed in for ordered parameter sort",
		"Failed to parse $size",
		"BadValue",
		"NoQueryExecutionPlans",
		"unable to find index for $geoNear query",
		"AuthenticationFailed", // In case of auth issues during testing
		"text index required for $text query",
		"geo index required for",
		"exceeded memory limit",
		"sort exceeded memory limit",
		"aggregation exceeded memory limit",
	}

	for _, expected := range expectedErrors {
		if strings.Contains(errStr, expected) {
			return true
		}
	}

	return false
}

// performUnoptimizedOperation demonstrates various common MongoDB developer mistakes
func (s *Service) performUnoptimizedOperation(ctx context.Context, client *mongo.Client) error {
	// Choose from various problematic query patterns
	queryType := rand.Intn(18)

	switch queryType {
	case 0:
		// No indexes with large collection scan
		return s.largeCollectionScan(ctx, client)
	case 1:
		// N+1 Query problem
		return s.nPlusOneQuery(ctx, client)
	case 2:
		// Inefficient regex queries
		return s.inefficientRegexQuery(ctx, client)
	case 3:
		// Missing compound indexes
		return s.missingCompoundIndex(ctx, client)
	case 4:
		// Large skip() operations
		return s.largeSkipOperation(ctx, client)
	case 5:
		// Retrieving entire documents when only fields needed
		return s.selectAllFields(ctx, client)
	case 6:
		// Inefficient array queries
		return s.inefficientArrayQuery(ctx, client)
	case 7:
		// Missing limits on large results
		return s.missingLimit(ctx, client)
	case 8:
		// Inefficient aggregation pipelines
		return s.inefficientAggregation(ctx, client)
	case 9:
		// Wrong data types in queries
		return s.wrongDataTypes(ctx, client)
	case 10:
		// Inefficient text search
		return s.inefficientTextSearch(ctx, client)
	case 11:
		// Unoptimized geospatial queries
		return s.inefficientGeoQuery(ctx, client)
	case 12:
		// Memory-intensive operations
		return s.memoryIntensiveOperation(ctx, client)
	case 13:
		// Inefficient date range queries
		return s.inefficientDateRangeQuery(ctx, client)
	case 14:
		// Large $in operations
		return s.largeInOperation(ctx, client)
	case 15:
		// Inefficient counting
		return s.inefficientCounting(ctx, client)
	case 16:
		// Multiple database round trips
		return s.multipleRoundTrips(ctx, client)
	default:
		// Insert test data
		return s.insertTestData(ctx, client)
	}
}

func (s *Service) largeCollectionScan(ctx context.Context, client *mongo.Client) error {
	// Bad: Full collection scan without indexes
	collection := client.Database("loadtest").Collection("users")

	filter := bson.M{
		"profile.age":       bson.M{"$gte": 25, "$lte": 35},
		"profile.city":      bson.M{"$regex": "New.*", "$options": "i"},
		"preferences.theme": "dark",
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) nPlusOneQuery(ctx context.Context, client *mongo.Client) error {
	// Bad: N+1 queries instead of aggregation
	usersCollection := client.Database("loadtest").Collection("users")
	ordersCollection := client.Database("loadtest").Collection("orders")

	// Get users (1 query)
	cursor, err := usersCollection.Find(ctx, bson.M{}, options.Find().SetLimit(10))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	// For each user, get their orders (N queries)
	for cursor.Next(ctx) {
		var user bson.M
		if err := cursor.Decode(&user); err != nil {
			continue
		}

		userID, ok := user["_id"]
		if !ok {
			continue
		}

		// This creates N additional queries!
		orderCursor, err := ordersCollection.Find(ctx, bson.M{"user_id": userID})
		if err != nil {
			continue
		}

		for orderCursor.Next(ctx) {
			var order bson.M
			orderCursor.Decode(&order)
		}
		orderCursor.Close(ctx)
	}
	return cursor.Err()
}

func (s *Service) inefficientRegexQuery(ctx context.Context, client *mongo.Client) error {
	// Bad: Regex without anchoring and case-insensitive searches
	collection := client.Database("loadtest").Collection("users")

	filter := bson.M{
		"$or": []bson.M{
			{"email": bson.M{"$regex": ".*gmail.*", "$options": "i"}},    // Leading wildcard
			{"username": bson.M{"$regex": ".*admin.*", "$options": "i"}}, // Leading wildcard
			{"profile.bio": bson.M{"$regex": ".*developer.*", "$options": "i"}},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) missingCompoundIndex(ctx context.Context, client *mongo.Client) error {
	// Bad: Query requiring compound index but only single field indexes exist
	collection := client.Database("loadtest").Collection("orders")

	filter := bson.M{
		"status":       "pending",
		"created_at":   bson.M{"$gte": time.Now().AddDate(0, 0, -30)},
		"total_amount": bson.M{"$gte": 100},
		"user_id":      bson.M{"$exists": true},
	}

	opts := options.Find().SetSort(bson.M{"created_at": -1, "total_amount": -1})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) largeSkipOperation(ctx context.Context, client *mongo.Client) error {
	// Bad: Large skip() for pagination instead of cursor-based pagination
	collection := client.Database("loadtest").Collection("users")

	skipCount := rand.Intn(10000) + 5000 // Large skip
	opts := options.Find().SetSkip(int64(skipCount)).SetLimit(20)

	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) selectAllFields(ctx context.Context, client *mongo.Client) error {
	// Bad: Retrieving entire large documents when only specific fields needed
	collection := client.Database("loadtest").Collection("users")

	// Should use projection to only get needed fields, but doesn't
	cursor, err := collection.Find(ctx, bson.M{"status": "active"})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M // Getting all fields instead of specific ones
		cursor.Decode(&result)

		// Only using a few fields but loaded everything
		_, _ = result["_id"], result["username"]
	}
	return cursor.Err()
}

func (s *Service) inefficientArrayQuery(ctx context.Context, client *mongo.Client) error {
	// Bad: Inefficient array queries without proper indexing
	collection := client.Database("loadtest").Collection("users")

	filter := bson.M{
		"$and": []bson.M{
			{"tags.0": bson.M{"$exists": true}},          // Checking array position
			{"tags": bson.M{"$size": bson.M{"$gte": 3}}}, // Bad: $size with range
			{"tags": bson.M{"$regex": ".*premium.*"}},    // Regex on array
			{"preferences.notifications.email": true},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) missingLimit(ctx context.Context, client *mongo.Client) error {
	// Bad: No limit on potentially large result set
	collection := client.Database("loadtest").Collection("audit_log")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) && count < 100 { // Artificial limit in application
		var result bson.M
		cursor.Decode(&result)
		count++
	}
	return cursor.Err()
}

func (s *Service) inefficientAggregation(ctx context.Context, client *mongo.Client) error {
	// Bad: Inefficient aggregation pipeline with unnecessary stages
	collection := client.Database("loadtest").Collection("orders")

	pipeline := []bson.M{
		// Bad: No $match early in pipeline
		{"$lookup": bson.M{
			"from":         "users",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "user_info",
		}},
		{"$unwind": "$user_info"}, // Expensive unwind before filtering
		{"$addFields": bson.M{
			"year":  bson.M{"$year": "$created_at"},
			"month": bson.M{"$month": "$created_at"},
			"day":   bson.M{"$dayOfMonth": "$created_at"},
		}},
		{"$match": bson.M{ // Should be much earlier
			"status":       "completed",
			"total_amount": bson.M{"$gte": 100},
		}},
		{"$group": bson.M{
			"_id": bson.M{
				"year":   "$year",
				"month":  "$month",
				"status": "$status",
			},
			"total_orders": bson.M{"$sum": 1},
			"avg_amount":   bson.M{"$avg": "$total_amount"},
			"max_amount":   bson.M{"$max": "$total_amount"},
		}},
		{"$sort": bson.M{"_id.year": -1, "_id.month": -1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) wrongDataTypes(ctx context.Context, client *mongo.Client) error {
	// Bad: Querying with wrong data types (string vs ObjectId, etc.)
	collection := client.Database("loadtest").Collection("orders")

	// Using string instead of ObjectId
	filter := bson.M{
		"user_id": fmt.Sprintf("%d", rand.Intn(1000)),       // String instead of ObjectId
		"amount":  fmt.Sprintf("%.2f", rand.Float64()*1000), // String instead of float
		"status":  1,                                        // Number instead of string
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) inefficientTextSearch(ctx context.Context, client *mongo.Client) error {
	// Bad: Using regex instead of text index for search
	collection := client.Database("loadtest").Collection("products")

	searchTerms := []string{"laptop", "phone", "tablet", "computer"}
	searchTerm := searchTerms[rand.Intn(len(searchTerms))]

	filter := bson.M{
		"$or": []bson.M{
			{"name": bson.M{"$regex": searchTerm, "$options": "i"}},
			{"description": bson.M{"$regex": searchTerm, "$options": "i"}},
			{"category": bson.M{"$regex": searchTerm, "$options": "i"}},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) inefficientGeoQuery(ctx context.Context, client *mongo.Client) error {
	// Bad: Geospatial query without proper 2dsphere index
	collection := client.Database("loadtest").Collection("locations")

	// Random location (approximately NYC)
	longitude := -74.0 + (rand.Float64()-0.5)*0.1
	latitude := 40.7 + (rand.Float64()-0.5)*0.1

	filter := bson.M{
		"location": bson.M{
			"$near": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{longitude, latitude},
				},
				"$maxDistance": 5000, // 5km
			},
		},
		"type": "restaurant",
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) memoryIntensiveOperation(ctx context.Context, client *mongo.Client) error {
	// Bad: Memory-intensive operations that could exceed memory limits
	collection := client.Database("loadtest").Collection("orders")

	pipeline := []bson.M{
		{"$unwind": "$items"}, // Potentially explodes document count
		{"$lookup": bson.M{
			"from":         "products",
			"localField":   "items.product_id",
			"foreignField": "_id",
			"as":           "product_info",
		}},
		{"$unwind": "$product_info"},
		{"$lookup": bson.M{ // Second lookup without limits
			"from":         "categories",
			"localField":   "product_info.category_id",
			"foreignField": "_id",
			"as":           "category_info",
		}},
		{"$group": bson.M{
			"_id":            "$product_info.category_id",
			"all_orders":     bson.M{"$push": "$$ROOT"}, // Accumulating entire documents
			"total_quantity": bson.M{"$sum": "$items.quantity"},
		}},
		{"$sort": bson.M{"total_quantity": -1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) inefficientDateRangeQuery(ctx context.Context, client *mongo.Client) error {
	// Bad: Date range queries with string comparison instead of Date objects
	collection := client.Database("loadtest").Collection("events")

	filter := bson.M{
		"$and": []bson.M{
			{"created_at_string": bson.M{"$gte": "2023-01-01"}}, // String comparison
			{"created_at_string": bson.M{"$lte": "2023-12-31"}},
			{"$expr": bson.M{
				"$eq": []interface{}{
					bson.M{"$dayOfWeek": "$created_at"}, 1, // Monday only
				},
			}},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) largeInOperation(ctx context.Context, client *mongo.Client) error {
	// Bad: Very large $in operation
	collection := client.Database("loadtest").Collection("users")

	// Create large array for $in operation
	largeInArray := make([]interface{}, 5000)
	for i := 0; i < 5000; i++ {
		largeInArray[i] = primitive.NewObjectID()
	}

	filter := bson.M{
		"_id": bson.M{"$in": largeInArray},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		cursor.Decode(&result)
	}
	return cursor.Err()
}

func (s *Service) inefficientCounting(ctx context.Context, client *mongo.Client) error {
	// Bad: Using find().count() instead of countDocuments for accuracy
	collection := client.Database("loadtest").Collection("orders")

	// First bad way: finding all then counting in application
	cursor, err := collection.Find(ctx, bson.M{"status": "pending"})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		count++
	}

	// Second bad way: estimated count when exact count needed
	estimatedCount, err := collection.EstimatedDocumentCount(ctx)
	if err != nil {
		return err
	}
	_ = estimatedCount

	return cursor.Err()
}

func (s *Service) multipleRoundTrips(ctx context.Context, client *mongo.Client) error {
	// Bad: Multiple separate queries instead of single aggregation
	usersCollection := client.Database("loadtest").Collection("users")
	ordersCollection := client.Database("loadtest").Collection("orders")

	// Query 1: Get user count
	userCount, err := usersCollection.CountDocuments(ctx, bson.M{"status": "active"})
	if err != nil {
		return err
	}

	// Query 2: Get average order amount
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":        nil,
			"avg_amount": bson.M{"$avg": "$total_amount"},
		}},
	}
	cursor, err := ordersCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	var avgResult bson.M
	if cursor.Next(ctx) {
		cursor.Decode(&avgResult)
	}
	cursor.Close(ctx)

	// Query 3: Get top categories
	categoryPipeline := []bson.M{
		{"$group": bson.M{
			"_id":   "$category",
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"count": -1}},
		{"$limit": 5},
	}
	categoryCursor, err := ordersCollection.Aggregate(ctx, categoryPipeline)
	if err != nil {
		return err
	}
	defer categoryCursor.Close(ctx)

	for categoryCursor.Next(ctx) {
		var result bson.M
		categoryCursor.Decode(&result)
	}

	// Using results (just to avoid unused variable warnings)
	_ = userCount

	return categoryCursor.Err()
}

func (s *Service) insertTestData(ctx context.Context, client *mongo.Client) error {
	// Insert test data for other operations
	collections := []string{"users", "orders", "products", "audit_log", "events", "locations"}
	collectionName := collections[rand.Intn(len(collections))]
	collection := client.Database("loadtest").Collection(collectionName)

	switch collectionName {
	case "users":
		doc := bson.M{
			"username": fmt.Sprintf("user_%d", rand.Intn(10000)),
			"email":    fmt.Sprintf("user%d@example.com", rand.Intn(10000)),
			"profile": bson.M{
				"age":  rand.Intn(70) + 18,
				"city": []string{"New York", "Los Angeles", "Chicago", "Houston"}[rand.Intn(4)],
				"bio":  fmt.Sprintf("I am a developer with %d years of experience", rand.Intn(20)),
			},
			"preferences": bson.M{
				"theme":         []string{"dark", "light"}[rand.Intn(2)],
				"notifications": bson.M{"email": rand.Intn(2) == 1},
			},
			"tags":       []string{"user", "active", "premium"}[:rand.Intn(3)+1],
			"created_at": time.Now().AddDate(0, 0, -rand.Intn(365)),
			"status":     []string{"active", "inactive", "pending"}[rand.Intn(3)],
		}
		_, err := collection.InsertOne(ctx, doc)
		return err

	case "orders":
		doc := bson.M{
			"user_id":      primitive.NewObjectID(),
			"order_number": fmt.Sprintf("ORD-%d", rand.Intn(100000)),
			"total_amount": rand.Float64() * 1000,
			"status":       []string{"pending", "processing", "shipped", "delivered", "cancelled"}[rand.Intn(5)],
			"created_at":   time.Now().AddDate(0, 0, -rand.Intn(90)),
			"items": []bson.M{
				{
					"product_id": primitive.NewObjectID(),
					"quantity":   rand.Intn(5) + 1,
					"price":      rand.Float64() * 100,
				},
			},
			"category": []string{"electronics", "clothing", "books", "home"}[rand.Intn(4)],
		}
		_, err := collection.InsertOne(ctx, doc)
		return err

	case "products":
		doc := bson.M{
			"name":        fmt.Sprintf("Product %d", rand.Intn(1000)),
			"description": []string{"This is a great product for laptop", "This is a great product for phone", "This is a great product for tablet", "This is a great product for computer"}[rand.Intn(4)],
			"category":    []string{"electronics", "clothing", "books"}[rand.Intn(3)],
			"price":       rand.Float64() * 500,
			"created_at":  time.Now().AddDate(0, 0, -rand.Intn(365)),
		}
		_, err := collection.InsertOne(ctx, doc)
		return err

	case "audit_log":
		doc := bson.M{
			"action":     []string{"create", "update", "delete"}[rand.Intn(3)],
			"table_name": []string{"users", "orders", "products"}[rand.Intn(3)],
			"record_id":  primitive.NewObjectID(),
			"user_id":    primitive.NewObjectID(),
			"timestamp":  time.Now().AddDate(0, 0, -rand.Intn(30)),
			"changes":    bson.M{"field": "value"},
		}
		_, err := collection.InsertOne(ctx, doc)
		return err

	case "events":
		doc := bson.M{
			"type":              []string{"click", "view", "purchase"}[rand.Intn(3)],
			"user_id":           primitive.NewObjectID(),
			"created_at":        time.Now().AddDate(0, 0, -rand.Intn(365)),
			"created_at_string": time.Now().AddDate(0, 0, -rand.Intn(365)).Format("2006-01-02"),
			"metadata":          bson.M{"page": "/products", "category": "electronics"},
		}
		_, err := collection.InsertOne(ctx, doc)
		return err

	case "locations":
		doc := bson.M{
			"name": fmt.Sprintf("Location %d", rand.Intn(1000)),
			"type": []string{"restaurant", "shop", "office"}[rand.Intn(3)],
			"location": bson.M{
				"type":        "Point",
				"coordinates": []float64{-74.0 + (rand.Float64()-0.5)*0.2, 40.7 + (rand.Float64()-0.5)*0.2},
			},
			"created_at": time.Now().AddDate(0, 0, -rand.Intn(365)),
		}
		_, err := collection.InsertOne(ctx, doc)
		return err
	}

	return nil
}

func (s *Service) SeedData(ctx context.Context) error {
	fmt.Println("Seeding MongoDB with test data...")

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(s.dsn))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB for seeding: %w", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("loadtest")

	// Check if data already exists
	usersCollection := db.Collection("users")
	userCount, err := usersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if userCount > 100 {
		fmt.Printf("MongoDB already has %d users, skipping seeding\n", userCount)
		return nil
	}

	fmt.Println("Inserting seed data for MongoDB...")

	// Seed users (1000 records)
	fmt.Println("  - Inserting users...")
	var userDocs []interface{}
	for i := 0; i < 1000; i++ {
		birthDate := time.Now().AddDate(-rand.Intn(60)-18, -rand.Intn(12), -rand.Intn(365))
		age := 2024 - birthDate.Year()

		userDoc := bson.M{
			"username": fmt.Sprintf("user_%04d", i),
			"email":    fmt.Sprintf("user%04d@example.com", i),
			"profile": bson.M{
				"age":  age,
				"city": []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego"}[rand.Intn(8)],
				"bio":  fmt.Sprintf("I am a developer with %d years of experience working in %s", rand.Intn(20), []string{"fintech", "healthcare", "education", "gaming", "e-commerce"}[rand.Intn(5)]),
			},
			"preferences": bson.M{
				"theme": []string{"dark", "light"}[rand.Intn(2)],
				"notifications": bson.M{
					"email": rand.Intn(2) == 1,
					"sms":   rand.Intn(2) == 1,
					"push":  rand.Intn(2) == 1,
				},
			},
			"tags":       []string{"user", "active", "premium", "beta", "subscriber"}[:rand.Intn(3)+1],
			"created_at": birthDate,
			"status":     []string{"active", "inactive", "pending", "suspended"}[rand.Intn(4)],
			"metadata": bson.M{
				"last_login":    time.Now().AddDate(0, 0, -rand.Intn(30)),
				"login_count":   rand.Intn(1000),
				"referral_code": fmt.Sprintf("REF_%d", rand.Intn(10000)),
			},
		}
		userDocs = append(userDocs, userDoc)

		// Insert in batches to avoid memory issues
		if len(userDocs) == 100 {
			_, err := usersCollection.InsertMany(ctx, userDocs)
			if err != nil {
				return fmt.Errorf("failed to insert user batch: %w", err)
			}
			userDocs = userDocs[:0] // Reset slice
		}
	}
	// Insert remaining documents
	if len(userDocs) > 0 {
		_, err := usersCollection.InsertMany(ctx, userDocs)
		if err != nil {
			return fmt.Errorf("failed to insert final user batch: %w", err)
		}
	}

	// Get user IDs for orders
	usersCursor, err := usersCollection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}
	var userIDs []primitive.ObjectID
	for usersCursor.Next(ctx) {
		var result bson.M
		usersCursor.Decode(&result)
		userIDs = append(userIDs, result["_id"].(primitive.ObjectID))
	}
	usersCursor.Close(ctx)

	// Seed orders (2000 records)
	fmt.Println("  - Inserting orders...")
	ordersCollection := db.Collection("orders")
	var orderDocs []interface{}
	var orderIDs []primitive.ObjectID

	for i := 0; i < 2000; i++ {
		orderID := primitive.NewObjectID()
		orderIDs = append(orderIDs, orderID)
		userID := userIDs[rand.Intn(len(userIDs))]
		orderDate := time.Now().AddDate(0, 0, -rand.Intn(365))

		// Create items array
		numItems := rand.Intn(5) + 1
		items := make([]bson.M, numItems)
		totalAmount := 0.0

		for j := 0; j < numItems; j++ {
			quantity := rand.Intn(3) + 1
			unitPrice := rand.Float64()*100 + 5
			itemTotal := float64(quantity) * unitPrice
			totalAmount += itemTotal

			items[j] = bson.M{
				"product_id":   primitive.NewObjectID(),
				"product_name": fmt.Sprintf("Product_%04d", rand.Intn(500)),
				"quantity":     quantity,
				"unit_price":   unitPrice,
				"total_price":  itemTotal,
			}
		}

		orderDoc := bson.M{
			"_id":          orderID,
			"user_id":      userID,
			"order_number": fmt.Sprintf("ORD-%06d", i),
			"total_amount": totalAmount,
			"status":       []string{"pending", "processing", "shipped", "delivered", "cancelled"}[rand.Intn(5)],
			"created_at":   orderDate,
			"items":        items,
			"category":     []string{"electronics", "clothing", "books", "home", "sports", "beauty"}[rand.Intn(6)],
			"shipping_address": bson.M{
				"street":  fmt.Sprintf("%d Main St", rand.Intn(9999)+1),
				"city":    []string{"New York", "Los Angeles", "Chicago", "Houston"}[rand.Intn(4)],
				"state":   []string{"NY", "CA", "IL", "TX"}[rand.Intn(4)],
				"zipcode": fmt.Sprintf("%05d", rand.Intn(99999)),
			},
		}
		orderDocs = append(orderDocs, orderDoc)

		// Insert in batches
		if len(orderDocs) == 100 {
			_, err := ordersCollection.InsertMany(ctx, orderDocs)
			if err != nil {
				return fmt.Errorf("failed to insert order batch: %w", err)
			}
			orderDocs = orderDocs[:0]
		}
	}
	if len(orderDocs) > 0 {
		_, err := ordersCollection.InsertMany(ctx, orderDocs)
		if err != nil {
			return fmt.Errorf("failed to insert final order batch: %w", err)
		}
	}

	// Seed products (500 records)
	fmt.Println("  - Inserting products...")
	productsCollection := db.Collection("products")
	var productDocs []interface{}

	categories := []string{"electronics", "clothing", "books", "home", "sports", "beauty"}
	productNames := []string{"laptop", "phone", "tablet", "computer", "monitor", "keyboard", "mouse", "headphones", "speaker", "camera"}

	for i := 0; i < 500; i++ {
		productDoc := bson.M{
			"name":        fmt.Sprintf("%s_%04d", productNames[rand.Intn(len(productNames))], i),
			"description": fmt.Sprintf("This is a great %s product for laptop phone tablet computer use", productNames[rand.Intn(len(productNames))]),
			"category":    categories[rand.Intn(len(categories))],
			"price":       rand.Float64()*500 + 10,
			"created_at":  time.Now().AddDate(0, 0, -rand.Intn(365)),
			"in_stock":    rand.Intn(2) == 1,
			"rating":      rand.Float64() * 5,
			"tags":        []string{"popular", "new", "featured", "sale"}[:rand.Intn(3)+1],
		}
		productDocs = append(productDocs, productDoc)

		if len(productDocs) == 100 {
			_, err := productsCollection.InsertMany(ctx, productDocs)
			if err != nil {
				return fmt.Errorf("failed to insert product batch: %w", err)
			}
			productDocs = productDocs[:0]
		}
	}
	if len(productDocs) > 0 {
		_, err := productsCollection.InsertMany(ctx, productDocs)
		if err != nil {
			return fmt.Errorf("failed to insert final product batch: %w", err)
		}
	}

	// Seed audit log (10000 records)
	fmt.Println("  - Inserting audit log entries...")
	auditCollection := db.Collection("audit_log")
	var auditDocs []interface{}

	for i := 0; i < 10000; i++ {
		auditDoc := bson.M{
			"action":     []string{"create", "update", "delete", "view"}[rand.Intn(4)],
			"table_name": []string{"users", "orders", "products"}[rand.Intn(3)],
			"record_id":  primitive.NewObjectID(),
			"user_id":    userIDs[rand.Intn(len(userIDs))],
			"timestamp":  time.Now().AddDate(0, 0, -rand.Intn(30)),
			"changes": bson.M{
				"field":     fmt.Sprintf("field_%d", rand.Intn(10)),
				"old_value": fmt.Sprintf("old_%d", rand.Intn(100)),
				"new_value": fmt.Sprintf("new_%d", rand.Intn(100)),
			},
			"ip_address": fmt.Sprintf("192.168.1.%d", rand.Intn(255)),
			"user_agent": "Mozilla/5.0 (compatible; LoadGenerator/1.0)",
		}
		auditDocs = append(auditDocs, auditDoc)

		if len(auditDocs) == 100 {
			_, err := auditCollection.InsertMany(ctx, auditDocs)
			if err != nil {
				return fmt.Errorf("failed to insert audit batch: %w", err)
			}
			auditDocs = auditDocs[:0]
		}
	}
	if len(auditDocs) > 0 {
		_, err := auditCollection.InsertMany(ctx, auditDocs)
		if err != nil {
			return fmt.Errorf("failed to insert final audit batch: %w", err)
		}
	}

	// Seed events (5000 records)
	fmt.Println("  - Inserting events...")
	eventsCollection := db.Collection("events")
	var eventDocs []interface{}

	for i := 0; i < 5000; i++ {
		eventDate := time.Now().AddDate(0, 0, -rand.Intn(365))
		eventDoc := bson.M{
			"type":              []string{"click", "view", "purchase", "login", "logout", "search"}[rand.Intn(6)],
			"user_id":           userIDs[rand.Intn(len(userIDs))],
			"created_at":        eventDate,
			"created_at_string": eventDate.Format("2006-01-02"),
			"metadata": bson.M{
				"page":     []string{"/products", "/categories", "/profile", "/cart", "/checkout"}[rand.Intn(5)],
				"category": []string{"electronics", "clothing", "books"}[rand.Intn(3)],
				"duration": rand.Intn(300), // seconds
			},
			"session_id": fmt.Sprintf("session_%d", rand.Intn(1000)),
		}
		eventDocs = append(eventDocs, eventDoc)

		if len(eventDocs) == 100 {
			_, err := eventsCollection.InsertMany(ctx, eventDocs)
			if err != nil {
				return fmt.Errorf("failed to insert event batch: %w", err)
			}
			eventDocs = eventDocs[:0]
		}
	}
	if len(eventDocs) > 0 {
		_, err := eventsCollection.InsertMany(ctx, eventDocs)
		if err != nil {
			return fmt.Errorf("failed to insert final event batch: %w", err)
		}
	}

	// Seed locations (1000 records for geo queries)
	fmt.Println("  - Inserting locations...")
	locationsCollection := db.Collection("locations")
	var locationDocs []interface{}

	for i := 0; i < 1000; i++ {
		// Random locations around major cities
		baseCoords := [][]float64{
			{-74.0, 40.7},  // NYC
			{-118.2, 34.0}, // LA
			{-87.6, 41.9},  // Chicago
			{-95.4, 29.8},  // Houston
		}
		baseCoord := baseCoords[rand.Intn(len(baseCoords))]
		longitude := baseCoord[0] + (rand.Float64()-0.5)*0.2
		latitude := baseCoord[1] + (rand.Float64()-0.5)*0.2

		locationDoc := bson.M{
			"name": fmt.Sprintf("Location_%04d", i),
			"type": []string{"restaurant", "shop", "office", "hotel", "gas_station"}[rand.Intn(5)],
			"location": bson.M{
				"type":        "Point",
				"coordinates": []float64{longitude, latitude},
			},
			"created_at": time.Now().AddDate(0, 0, -rand.Intn(365)),
			"rating":     rand.Float64() * 5,
			"address": bson.M{
				"street": fmt.Sprintf("%d %s St", rand.Intn(9999)+1, []string{"Main", "First", "Second", "Oak", "Pine"}[rand.Intn(5)]),
				"city":   []string{"New York", "Los Angeles", "Chicago", "Houston"}[rand.Intn(4)],
			},
		}
		locationDocs = append(locationDocs, locationDoc)

		if len(locationDocs) == 100 {
			_, err := locationsCollection.InsertMany(ctx, locationDocs)
			if err != nil {
				return fmt.Errorf("failed to insert location batch: %w", err)
			}
			locationDocs = locationDocs[:0]
		}
	}
	if len(locationDocs) > 0 {
		_, err := locationsCollection.InsertMany(ctx, locationDocs)
		if err != nil {
			return fmt.Errorf("failed to insert final location batch: %w", err)
		}
	}

	fmt.Println("MongoDB data seeding completed successfully")
	return nil
}
