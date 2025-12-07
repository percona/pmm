package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"db-loadgen/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

type ProgressTracker interface {
	AddPostgresOp()
	AddPostgresError()
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

func (s *Service) RunMigrations() error {
	fmt.Println("Running PostgreSQL migrations...")

	db, err := sql.Open("postgres", s.dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL driver: %w", err)
	}

	// Create embedded filesystem source
	sourceDriver, err := iofs.New(migrations.GetPostgresMigrations(), ".")
	if err != nil {
		return fmt.Errorf("failed to create embedded migration source: %w", err)
	}

	// Create migrate instance with embedded source
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance with embedded source: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run PostgreSQL migrations: %w", err)
	}

	fmt.Println("PostgreSQL migrations completed successfully")
	return nil
}

func (s *Service) SeedData() error {
	fmt.Println("Seeding PostgreSQL with test data...")

	db, err := sql.Open("postgres", s.dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL for seeding: %w", err)
	}
	defer db.Close()

	// Check if data already exists
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if userCount > 100 {
		fmt.Printf("PostgreSQL already has %d users, skipping seeding\n", userCount)
		return nil
	}

	fmt.Println("Inserting seed data for PostgreSQL...")

	// Seed users (1000 records)
	fmt.Println("  - Inserting users...")
	userStmt, err := db.Prepare(`INSERT INTO users (username, email, first_name, last_name, birth_date, profile_data) VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		return fmt.Errorf("failed to prepare user insert: %w", err)
	}
	defer userStmt.Close()

	for i := 0; i < 1000; i++ {
		birthDate := time.Now().AddDate(-rand.Intn(60)-18, -rand.Intn(12), -rand.Intn(365))
		profileData := fmt.Sprintf(`{"theme": "%s", "notifications": %t, "age": %d, "interests": ["tech", "sports", "music"], "settings": {"language": "en", "timezone": "UTC"}}`,
			[]string{"dark", "light"}[rand.Intn(2)],
			rand.Intn(2) == 1,
			2024-birthDate.Year())

		_, err = userStmt.Exec(
			fmt.Sprintf("user_%04d", i),
			fmt.Sprintf("user%04d@example.com", i),
			fmt.Sprintf("FirstName%d", rand.Intn(100)),
			fmt.Sprintf("LastName%d", rand.Intn(100)),
			birthDate,
			profileData,
		)
		if err != nil {
			return fmt.Errorf("failed to insert user %d: %w", i, err)
		}
	}

	// Seed categories (50 records with hierarchy)
	fmt.Println("  - Inserting categories...")
	categoryStmt, err := db.Prepare(`INSERT INTO categories (name, parent_id, description, sort_order) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		return fmt.Errorf("failed to prepare category insert: %w", err)
	}
	defer categoryStmt.Close()

	// Root categories
	for i := 0; i < 10; i++ {
		_, err = categoryStmt.Exec(
			fmt.Sprintf("Category_%02d", i),
			nil,
			fmt.Sprintf("Root category %d description", i),
			i,
		)
		if err != nil {
			return fmt.Errorf("failed to insert category %d: %w", i, err)
		}
	}

	// Sub categories
	for i := 10; i < 50; i++ {
		parentID := (i % 10) + 1 // Reference to root categories
		_, err = categoryStmt.Exec(
			fmt.Sprintf("Subcategory_%02d", i),
			parentID,
			fmt.Sprintf("Subcategory %d description", i),
			i,
		)
		if err != nil {
			return fmt.Errorf("failed to insert subcategory %d: %w", i, err)
		}
	}

	// Seed orders (2000 records)
	fmt.Println("  - Inserting orders...")
	orderStmt, err := db.Prepare(`INSERT INTO orders (user_id, order_number, total_amount, status, order_date) VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		return fmt.Errorf("failed to prepare order insert: %w", err)
	}
	defer orderStmt.Close()

	for i := 0; i < 2000; i++ {
		userID := rand.Intn(1000) + 1
		orderDate := time.Now().AddDate(0, 0, -rand.Intn(365))
		statuses := []string{"pending", "processing", "shipped", "delivered", "cancelled"}
		status := statuses[rand.Intn(len(statuses))]

		_, err = orderStmt.Exec(
			userID,
			fmt.Sprintf("ORD-%06d", i),
			rand.Float64()*1000+10,
			status,
			orderDate,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order %d: %w", i, err)
		}
	}

	// Seed order items (5000 records)
	fmt.Println("  - Inserting order items...")
	itemStmt, err := db.Prepare(`INSERT INTO order_items (order_id, product_name, quantity, unit_price, total_price) VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		return fmt.Errorf("failed to prepare order item insert: %w", err)
	}
	defer itemStmt.Close()

	for i := 0; i < 5000; i++ {
		orderID := rand.Intn(2000) + 1
		quantity := rand.Intn(5) + 1
		unitPrice := rand.Float64()*100 + 5
		totalPrice := float64(quantity) * unitPrice

		_, err = itemStmt.Exec(
			orderID,
			fmt.Sprintf("Product_%04d", rand.Intn(500)),
			quantity,
			unitPrice,
			totalPrice,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order item %d: %w", i, err)
		}
	}

	// Seed audit log (10000 records for large table scans)
	fmt.Println("  - Inserting audit log entries...")
	auditStmt, err := db.Prepare(`INSERT INTO audit_log (table_name, operation_type, record_id, user_id, session_id, ip_address) VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		return fmt.Errorf("failed to prepare audit insert: %w", err)
	}
	defer auditStmt.Close()

	operationTypes := []string{"INSERT", "UPDATE", "DELETE"}
	tableNames := []string{"users", "orders", "categories", "order_items"}

	for i := 0; i < 10000; i++ {
		tableName := tableNames[rand.Intn(len(tableNames))]
		operationType := operationTypes[rand.Intn(len(operationTypes))]
		recordID := rand.Intn(1000) + 1
		userID := rand.Intn(1000) + 1

		_, err = auditStmt.Exec(
			tableName,
			operationType,
			recordID,
			userID,
			fmt.Sprintf("session_%d", rand.Intn(1000)),
			fmt.Sprintf("192.168.1.%d", rand.Intn(255)),
		)
		if err != nil {
			return fmt.Errorf("failed to insert audit log %d: %w", i, err)
		}
	}

	fmt.Println("PostgreSQL data seeding completed successfully")
	return nil
}

func (s *Service) StartLoad(ctx context.Context, wg *sync.WaitGroup) {
	fmt.Printf("Starting PostgreSQL load with %d workers\n", s.workers)

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go s.worker(ctx, wg, i)
	}
}

func (s *Service) worker(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()

	db, err := sql.Open("postgres", s.dsn)
	if err != nil {
		log.Printf("PostgreSQL worker %d: failed to connect: %v", id, err)
		return
	}
	defer db.Close()

	operations := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := s.performUnoptimizedOperation(db); err != nil {
				if s.tracker != nil {
					s.tracker.AddPostgresError()
				}
			} else {
				operations++
				if s.tracker != nil {
					s.tracker.AddPostgresOp()
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// performUnoptimizedOperation demonstrates various common developer mistakes in PostgreSQL
func (s *Service) performUnoptimizedOperation(db *sql.DB) error {
	// Choose from various problematic query patterns
	queryType := rand.Intn(22)

	switch queryType {
	case 0:
		// SELECT * (bad practice)
		return s.selectStarQuery(db)
	case 1:
		// N+1 Query problem
		return s.nPlusOneQuery(db)
	case 2:
		// Missing WHERE clause
		return s.missingWhereClause(db)
	case 3:
		// Function in WHERE clause
		return s.functionInWhereClause(db)
	case 4:
		// LIKE with leading wildcard
		return s.leadingWildcardLike(db)
	case 5:
		// Inefficient subquery instead of JOIN
		return s.inefficientSubquery(db)
	case 6:
		// Missing LIMIT
		return s.missingLimit(db)
	case 7:
		// Unnecessary DISTINCT
		return s.unnecessaryDistinct(db)
	case 8:
		// ORDER BY without proper index
		return s.inefficientOrderBy(db)
	case 9:
		// Complex CASE statement
		return s.complexCaseStatement(db)
	case 10:
		// OR conditions instead of UNION
		return s.inefficientOrConditions(db)
	case 11:
		// Inefficient aggregation
		return s.inefficientAggregation(db)
	case 12:
		// Self-join without proper indexes
		return s.inefficientSelfJoin(db)
	case 13:
		// JSONB operations without indexes
		return s.inefficientJsonbQuery(db)
	case 14:
		// Correlated subquery
		return s.correlatedSubquery(db)
	case 15:
		// Multiple table scan
		return s.multipleTableScan(db)
	case 16:
		// Non-sargable date operations
		return s.nonSargableDateOperations(db)
	case 17:
		// Large IN clause
		return s.largeInClause(db)
	case 18:
		// Inefficient GROUP BY
		return s.inefficientGroupBy(db)
	case 19:
		// Window function misuse
		return s.inefficientWindowFunction(db)
	case 20:
		// Recursive CTE without limits
		return s.inefficientRecursiveCTE(db)
	default:
		// Insert test data
		return s.insertTestData(db)
	}
}

func (s *Service) selectStarQuery(db *sql.DB) error {
	// Bad: SELECT * with multiple JOINs
	rows, err := db.Query(`
		SELECT * FROM users u 
		JOIN orders o ON u.id = o.user_id 
		JOIN order_items oi ON o.id = oi.order_id
		ORDER BY RANDOM() LIMIT 5`)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		rows.Scan(valuePtrs...)
	}
	return rows.Err()
}

func (s *Service) nPlusOneQuery(db *sql.DB) error {
	// Classic N+1 problem
	userRows, err := db.Query("SELECT id, username FROM users LIMIT 10")
	if err != nil {
		return err
	}
	defer userRows.Close()

	for userRows.Next() {
		var userID int
		var username string
		if err := userRows.Scan(&userID, &username); err != nil {
			continue
		}

		// N additional queries
		orderRows, err := db.Query("SELECT id, total_amount FROM orders WHERE user_id = $1", userID)
		if err != nil {
			continue
		}

		for orderRows.Next() {
			var orderID int
			var totalAmount float64
			orderRows.Scan(&orderID, &totalAmount)
		}
		orderRows.Close()
	}
	return userRows.Err()
}

func (s *Service) missingWhereClause(db *sql.DB) error {
	// Bad: Full table scan
	rows, err := db.Query(`
		SELECT table_name, operation_type, COUNT(*) 
		FROM audit_log 
		GROUP BY table_name, operation_type 
		ORDER BY COUNT(*) DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, operationType string
		var count int
		rows.Scan(&tableName, &operationType, &count)
	}
	return rows.Err()
}

func (s *Service) functionInWhereClause(db *sql.DB) error {
	// Bad: Functions prevent index usage
	rows, err := db.Query(`
		SELECT id, username, email 
		FROM users 
		WHERE EXTRACT(YEAR FROM created_at) = 2023 
		AND UPPER(first_name) = 'JOHN'
		AND LENGTH(username) > 5`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var username, email string
		rows.Scan(&id, &username, &email)
	}
	return rows.Err()
}

func (s *Service) leadingWildcardLike(db *sql.DB) error {
	// Bad: Leading wildcards prevent index usage
	rows, err := db.Query(`
		SELECT id, username, email 
		FROM users 
		WHERE email ILIKE '%@gmail.com' 
		OR username ILIKE '%admin%'
		OR first_name ILIKE '%john%'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var username, email string
		rows.Scan(&id, &username, &email)
	}
	return rows.Err()
}

func (s *Service) inefficientSubquery(db *sql.DB) error {
	// Bad: Multiple subqueries instead of JOINs
	rows, err := db.Query(`
		SELECT u.id, u.username,
			(SELECT COUNT(*) FROM orders WHERE user_id = u.id) as order_count,
			(SELECT COALESCE(MAX(total_amount), 0) FROM orders WHERE user_id = u.id) as max_order,
			(SELECT MIN(order_date) FROM orders WHERE user_id = u.id) as first_order,
			(SELECT AVG(total_amount) FROM orders WHERE user_id = u.id) as avg_order
		FROM users u 
		WHERE u.is_active = true`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, orderCount int
		var username string
		var maxOrder, avgOrder sql.NullFloat64
		var firstOrder sql.NullTime
		rows.Scan(&id, &username, &orderCount, &maxOrder, &firstOrder, &avgOrder)
	}
	return rows.Err()
}

func (s *Service) missingLimit(db *sql.DB) error {
	// Bad: No LIMIT on large result set
	rows, err := db.Query(`
		SELECT al.*, u.username 
		FROM audit_log al 
		LEFT JOIN users u ON al.user_id = u.id 
		ORDER BY al.timestamp DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() && count < 100 {
		var id, recordID, userID sql.NullInt32
		var tableName, sessionID, ipAddress string
		var operationType sql.NullString
		var timestamp time.Time
		var username sql.NullString
		rows.Scan(&id, &tableName, &operationType, &recordID, &userID, &timestamp, &sessionID, &ipAddress, &username)
		count++
	}
	return rows.Err()
}

func (s *Service) unnecessaryDistinct(db *sql.DB) error {
	// Bad: DISTINCT on already unique columns
	rows, err := db.Query(`
		SELECT DISTINCT u.id, u.username, u.email 
		FROM users u 
		WHERE u.created_at > NOW() - INTERVAL '30 days'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var username, email string
		rows.Scan(&id, &username, &email)
	}
	return rows.Err()
}

func (s *Service) inefficientOrderBy(db *sql.DB) error {
	// Bad: ORDER BY on non-indexed columns
	rows, err := db.Query(`
		SELECT u.id, u.first_name, u.last_name, u.birth_date 
		FROM users u 
		ORDER BY u.last_name, u.first_name, u.birth_date 
		LIMIT 20`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var firstName, lastName sql.NullString
		var birthDate sql.NullTime
		rows.Scan(&id, &firstName, &lastName, &birthDate)
	}
	return rows.Err()
}

func (s *Service) complexCaseStatement(db *sql.DB) error {
	// Bad: Complex nested CASE statements
	rows, err := db.Query(`
		SELECT u.id, u.username,
			CASE 
				WHEN u.birth_date IS NULL THEN 'Unknown Age'
				WHEN EXTRACT(YEAR FROM AGE(u.birth_date)) < 18 THEN 'Minor'
				WHEN EXTRACT(YEAR FROM AGE(u.birth_date)) BETWEEN 18 AND 25 THEN 'Young Adult'
				WHEN EXTRACT(YEAR FROM AGE(u.birth_date)) BETWEEN 26 AND 40 THEN 'Adult'
				WHEN EXTRACT(YEAR FROM AGE(u.birth_date)) BETWEEN 41 AND 60 THEN 'Middle Age'
				ELSE 'Senior'
			END as age_category,
			CASE 
				WHEN (SELECT COUNT(*) FROM orders WHERE user_id = u.id) = 0 THEN 'No Orders'
				WHEN (SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE user_id = u.id) < 100 THEN 'Low Value'
				WHEN (SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE user_id = u.id) < 1000 THEN 'Medium Value'
				ELSE 'High Value'
			END as customer_tier
		FROM users u`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var username, ageCategory, customerTier string
		rows.Scan(&id, &username, &ageCategory, &customerTier)
	}
	return rows.Err()
}

func (s *Service) inefficientOrConditions(db *sql.DB) error {
	// Bad: Multiple OR conditions instead of optimized queries
	rows, err := db.Query(`
		SELECT u.id, u.username, u.email 
		FROM users u 
		WHERE u.username ILIKE 'admin%' 
		   OR u.email ILIKE '%@company.com' 
		   OR u.first_name = ANY(ARRAY['John', 'Jane', 'Bob', 'Alice'])
		   OR u.last_name = ANY(ARRAY['Smith', 'Johnson', 'Brown', 'Davis'])
		   OR EXTRACT(DOW FROM u.created_at) IN (0, 6)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var username, email string
		rows.Scan(&id, &username, &email)
	}
	return rows.Err()
}

func (s *Service) inefficientAggregation(db *sql.DB) error {
	// Bad: Complex aggregation without proper indexes
	rows, err := db.Query(`
		SELECT 
			DATE(al.timestamp) as log_date,
			al.table_name,
			COUNT(*) as operation_count,
			COUNT(DISTINCT al.user_id) as unique_users,
			AVG(al.record_id) as avg_record_id,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY al.record_id) as median_record_id
		FROM audit_log al 
		WHERE al.timestamp >= NOW() - INTERVAL '7 days'
		GROUP BY DATE(al.timestamp), al.table_name 
		HAVING COUNT(*) > 1
		ORDER BY operation_count DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var logDate time.Time
		var tableName string
		var operationCount, uniqueUsers int
		var avgRecordID, medianRecordID float64
		rows.Scan(&logDate, &tableName, &operationCount, &uniqueUsers, &avgRecordID, &medianRecordID)
	}
	return rows.Err()
}

func (s *Service) inefficientSelfJoin(db *sql.DB) error {
	// Bad: Deep self-join without proper indexing
	rows, err := db.Query(`
		SELECT 
			parent.id as parent_id,
			parent.name as parent_name,
			child.id as child_id,
			child.name as child_name,
			grandchild.name as grandchild_name,
			COUNT(great_grandchild.id) as great_grandchild_count
		FROM categories parent
		LEFT JOIN categories child ON parent.id = child.parent_id
		LEFT JOIN categories grandchild ON child.id = grandchild.parent_id
		LEFT JOIN categories great_grandchild ON grandchild.id = great_grandchild.parent_id
		GROUP BY parent.id, parent.name, child.id, child.name, grandchild.name
		ORDER BY parent.sort_order, child.sort_order`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var parentID, childID sql.NullInt32
		var parentName, childName, grandchildName sql.NullString
		var greatGrandchildCount int
		rows.Scan(&parentID, &parentName, &childID, &childName, &grandchildName, &greatGrandchildCount)
	}
	return rows.Err()
}

func (s *Service) inefficientJsonbQuery(db *sql.DB) error {
	// Bad: JSONB operations without GIN indexes
	rows, err := db.Query(`
		SELECT u.id, u.username, u.profile_data
		FROM users u 
		WHERE u.profile_data->>'theme' = 'dark'
		   OR (u.profile_data->'settings'->>'notifications')::boolean = true
		   OR jsonb_array_length(u.profile_data->'tags') > 3
		   OR u.profile_data ? 'preferences'
		   OR u.profile_data @> '{"type": "premium"}'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var username string
		var profileData sql.NullString
		rows.Scan(&id, &username, &profileData)
	}
	return rows.Err()
}

func (s *Service) correlatedSubquery(db *sql.DB) error {
	// Bad: Correlated subquery executes for each row
	rows, err := db.Query(`
		SELECT u.id, u.username,
			(SELECT COUNT(*) 
			 FROM orders o 
			 WHERE o.user_id = u.id AND o.status = 'delivered') as delivered_orders,
			(SELECT COALESCE(AVG(oi.total_price), 0)
			 FROM orders o2 
			 JOIN order_items oi ON o2.id = oi.order_id 
			 WHERE o2.user_id = u.id) as avg_item_price,
			(SELECT STRING_AGG(DISTINCT o3.status::text, ', ')
			 FROM orders o3 
			 WHERE o3.user_id = u.id) as all_statuses
		FROM users u 
		WHERE u.is_active = true`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, deliveredOrders int
		var username string
		var avgItemPrice sql.NullFloat64
		var allStatuses sql.NullString
		rows.Scan(&id, &username, &deliveredOrders, &avgItemPrice, &allStatuses)
	}
	return rows.Err()
}

func (s *Service) multipleTableScan(db *sql.DB) error {
	// Bad: Multiple full table scans
	rows, err := db.Query(`
		SELECT 
			(SELECT COUNT(*) FROM users WHERE is_active = true) as active_users,
			(SELECT COUNT(*) FROM orders WHERE status = 'pending') as pending_orders,
			(SELECT COUNT(*) FROM audit_log WHERE DATE(timestamp) = CURRENT_DATE) as todays_logs,
			(SELECT AVG(total_amount) FROM orders) as avg_order_amount,
			(SELECT COUNT(DISTINCT table_name) FROM audit_log) as audited_tables`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var activeUsers, pendingOrders, todaysLogs, auditedTables int
		var avgOrderAmount float64
		rows.Scan(&activeUsers, &pendingOrders, &todaysLogs, &avgOrderAmount, &auditedTables)
	}
	return rows.Err()
}

func (s *Service) nonSargableDateOperations(db *sql.DB) error {
	// Bad: Non-sargable date operations
	rows, err := db.Query(`
		SELECT o.id, o.order_number, o.total_amount 
		FROM orders o 
		WHERE EXTRACT(MONTH FROM o.order_date) = 12 
		  AND EXTRACT(YEAR FROM o.order_date) = 2023
		  AND EXTRACT(DOW FROM o.order_date) IN (0, 6)
		  AND DATE_PART('hour', o.order_date) BETWEEN 9 AND 17`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var orderNumber string
		var totalAmount float64
		rows.Scan(&id, &orderNumber, &totalAmount)
	}
	return rows.Err()
}

func (s *Service) largeInClause(db *sql.DB) error {
	// Bad: Very large IN clause
	inValues := make([]interface{}, 1000)
	placeholders := ""
	for i := 0; i < 1000; i++ {
		if i > 0 {
			placeholders += ","
		}
		placeholders += fmt.Sprintf("$%d", i+1)
		inValues[i] = rand.Intn(10000)
	}

	query := fmt.Sprintf(`
		SELECT u.id, u.username 
		FROM users u 
		WHERE u.id IN (%s)`, placeholders)

	rows, err := db.Query(query, inValues...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var username string
		rows.Scan(&id, &username)
	}
	return rows.Err()
}

func (s *Service) inefficientGroupBy(db *sql.DB) error {
	// Bad: GROUP BY with functions and no supporting indexes
	rows, err := db.Query(`
		SELECT 
			EXTRACT(YEAR FROM o.order_date) as order_year,
			EXTRACT(MONTH FROM o.order_date) as order_month,
			o.status,
			COUNT(*) as order_count,
			SUM(o.total_amount) as total_revenue,
			AVG(o.total_amount) as avg_order_value,
			COUNT(DISTINCT o.user_id) as unique_customers,
			STDDEV(o.total_amount) as amount_stddev
		FROM orders o 
		GROUP BY EXTRACT(YEAR FROM o.order_date), EXTRACT(MONTH FROM o.order_date), o.status 
		HAVING COUNT(*) > 1
		ORDER BY order_year DESC, order_month DESC, total_revenue DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderYear, orderMonth, orderCount, uniqueCustomers int
		var status string
		var totalRevenue, avgOrderValue, amountStddev sql.NullFloat64
		rows.Scan(&orderYear, &orderMonth, &status, &orderCount, &totalRevenue, &avgOrderValue, &uniqueCustomers, &amountStddev)
	}
	return rows.Err()
}

func (s *Service) inefficientWindowFunction(db *sql.DB) error {
	// Bad: Window functions without proper partitioning
	rows, err := db.Query(`
		SELECT 
			u.id,
			u.username,
			ROW_NUMBER() OVER (ORDER BY u.created_at) as user_number,
			DENSE_RANK() OVER (ORDER BY (SELECT COUNT(*) FROM orders WHERE user_id = u.id)) as order_count_rank,
			LAG(u.created_at, 1) OVER (ORDER BY u.created_at) as prev_user_created,
			LEAD(u.created_at, 1) OVER (ORDER BY u.created_at) as next_user_created,
			(SELECT AVG(total_amount) FROM orders WHERE user_id <= u.id) as running_avg_order
		FROM users u`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, userNumber, orderCountRank int
		var username string
		var prevCreated, nextCreated sql.NullTime
		var runningAvg sql.NullFloat64
		rows.Scan(&id, &username, &userNumber, &orderCountRank, &prevCreated, &nextCreated, &runningAvg)
	}
	return rows.Err()
}

func (s *Service) inefficientRecursiveCTE(db *sql.DB) error {
	// Bad: Recursive CTE without proper termination conditions
	rows, err := db.Query(`
		WITH RECURSIVE category_hierarchy AS (
			SELECT id, name, parent_id, 0 as level, ARRAY[id] as path
			FROM categories 
			WHERE parent_id IS NULL
			
			UNION ALL
			
			SELECT c.id, c.name, c.parent_id, ch.level + 1, ch.path || c.id
			FROM categories c
			JOIN category_hierarchy ch ON c.parent_id = ch.id
			WHERE NOT c.id = ANY(ch.path) -- Prevent infinite loops
		)
		SELECT ch.*, 
			(SELECT COUNT(*) FROM categories WHERE parent_id = ch.id) as direct_children,
			(SELECT STRING_AGG(name, ' > ') FROM categories WHERE id = ANY(ch.path)) as full_path
		FROM category_hierarchy ch
		ORDER BY level, name`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, parentID, level, directChildren int
		var name, fullPath string
		rows.Scan(&id, &name, &parentID, &level, &directChildren, &fullPath)
	}
	return rows.Err()
}

func (s *Service) insertTestData(db *sql.DB) error {
	// Insert test data for other queries
	queries := []string{
		`INSERT INTO users (username, email, first_name, last_name, birth_date, profile_data) VALUES 
		 ($1, $2, $3, $4, $5, $6)`,
		`INSERT INTO orders (user_id, order_number, total_amount, status) VALUES 
		 ((SELECT id FROM users ORDER BY RANDOM() LIMIT 1), $1, $2, $3)`,
		`INSERT INTO audit_log (table_name, operation_type, record_id, user_id, session_id, ip_address) VALUES 
		 ($1, $2, $3, $4, $5, $6)`,
		`INSERT INTO categories (name, parent_id, description, sort_order) VALUES 
		 ($1, $2, $3, $4)`,
	}

	queryIndex := rand.Intn(len(queries))

	switch queryIndex {
	case 0: // Users
		_, err := db.Exec(queries[0],
			fmt.Sprintf("user_%d", rand.Intn(10000)),
			fmt.Sprintf("user%d@example.com", rand.Intn(10000)),
			fmt.Sprintf("FirstName%d", rand.Intn(100)),
			fmt.Sprintf("LastName%d", rand.Intn(100)),
			time.Now().AddDate(-rand.Intn(50), -rand.Intn(12), -rand.Intn(365)),
			`{"theme": "dark", "notifications": true, "tags": ["user", "active"]}`)
		return err
	case 1: // Orders
		_, err := db.Exec(queries[1],
			fmt.Sprintf("ORD-%d", rand.Intn(100000)),
			rand.Float64()*1000,
			[]string{"pending", "processing", "shipped", "delivered", "cancelled"}[rand.Intn(5)])
		return err
	case 2: // Audit log
		_, err := db.Exec(queries[2],
			[]string{"users", "orders", "categories"}[rand.Intn(3)],
			[]string{"INSERT", "UPDATE", "DELETE"}[rand.Intn(3)],
			rand.Intn(1000),
			rand.Intn(100),
			fmt.Sprintf("session_%d", rand.Intn(1000)),
			fmt.Sprintf("192.168.1.%d", rand.Intn(255)))
		return err
	case 3: // Categories
		_, err := db.Exec(queries[3],
			fmt.Sprintf("Category%d", rand.Intn(1000)),
			nil, // No parent for simplicity
			fmt.Sprintf("Description for category %d", rand.Intn(1000)),
			rand.Intn(100))
		return err
	}
	return nil
}
