package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"db-loadgen/service/mongo"
	"db-loadgen/service/mysql"
	"db-loadgen/service/postgres"
)

type Config struct {
	MySQLDSN    string
	PostgresDSN string
	MongoDSN    string
	Duration    time.Duration
	Workers     int
}

type ProgressTracker struct {
	mysqlOps       int64
	postgresOps    int64
	mongoOps       int64
	mysqlErrors    int64
	postgresErrors int64
	mongoErrors    int64
	startTime      time.Time
	mu             sync.RWMutex
}

func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		startTime: time.Now(),
	}
}

func (pt *ProgressTracker) AddMySQLOp() {
	atomic.AddInt64(&pt.mysqlOps, 1)
}

func (pt *ProgressTracker) AddPostgresOp() {
	atomic.AddInt64(&pt.postgresOps, 1)
}

func (pt *ProgressTracker) AddMongoOp() {
	atomic.AddInt64(&pt.mongoOps, 1)
}

func (pt *ProgressTracker) AddMySQLError() {
	atomic.AddInt64(&pt.mysqlErrors, 1)
}

func (pt *ProgressTracker) AddPostgresError() {
	atomic.AddInt64(&pt.postgresErrors, 1)
}

func (pt *ProgressTracker) AddMongoError() {
	atomic.AddInt64(&pt.mongoErrors, 1)
}

func (pt *ProgressTracker) GetStats() (mysqlOps, postgresOps, mongoOps, mysqlErrors, postgresErrors, mongoErrors int64, elapsed time.Duration) {
	return atomic.LoadInt64(&pt.mysqlOps),
		atomic.LoadInt64(&pt.postgresOps),
		atomic.LoadInt64(&pt.mongoOps),
		atomic.LoadInt64(&pt.mysqlErrors),
		atomic.LoadInt64(&pt.postgresErrors),
		atomic.LoadInt64(&pt.mongoErrors),
		time.Since(pt.startTime)
}

func (pt *ProgressTracker) DisplayProgress(ctx context.Context, config Config) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Clear screen and hide cursor
	fmt.Print("\033[2J\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor on exit

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mysqlOps, postgresOps, mongoOps, mysqlErrors, postgresErrors, mongoErrors, elapsed := pt.GetStats()

			// Move cursor to top-left
			fmt.Print("\033[H")

			fmt.Printf("Database Load Generator - Running for %v\n", elapsed.Truncate(time.Second))
			if config.Duration > 0 {
				fmt.Printf("Target Duration: %v | Workers per DB: %d\n", config.Duration, config.Workers)
			} else {
				fmt.Printf("Running indefinitely (Ctrl+C to stop) | Workers per DB: %d\n", config.Workers)
			}
			fmt.Println(strings.Repeat("─", 80))

			if config.MySQLDSN != "" {
				rate := float64(mysqlOps) / elapsed.Seconds()
				fmt.Printf("mysql      │ Operations: %-8d │ Errors: %-6d │ Rate: %6.1f ops/s\n",
					mysqlOps, mysqlErrors, rate)
			}

			if config.PostgresDSN != "" {
				rate := float64(postgresOps) / elapsed.Seconds()
				fmt.Printf("postgres   │ Operations: %-8d │ Errors: %-6d │ Rate: %6.1f ops/s\n",
					postgresOps, postgresErrors, rate)
			}

			if config.MongoDSN != "" {
				rate := float64(mongoOps) / elapsed.Seconds()
				fmt.Printf("mongodb    │ Operations: %-8d │ Errors: %-6d │ Rate: %6.1f ops/s\n",
					mongoOps, mongoErrors, rate)
			}

			totalOps := mysqlOps + postgresOps + mongoOps
			totalErrors := mysqlErrors + postgresErrors + mongoErrors
			totalRate := float64(totalOps) / elapsed.Seconds()

			fmt.Println(strings.Repeat("─", 80))
			fmt.Printf("TOTAL      │ Operations: %-8d │ Errors: %-6d │ Rate: %6.1f ops/s\n",
				totalOps, totalErrors, totalRate)

			// Clear rest of screen
			fmt.Print("\033[J")
		}
	}
}

type LoadGenerator struct {
	config  Config
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	tracker *ProgressTracker
}

func main() {
	var config Config
	var durationStr string

	flag.StringVar(&config.MySQLDSN, "mysql-dsn", "", "MySQL connection string (e.g., user:password@tcp(localhost:3306)/dbname)")
	flag.StringVar(&config.PostgresDSN, "postgres-dsn", "", "PostgreSQL connection string (e.g., postgres://user:password@localhost/dbname?sslmode=disable)")
	flag.StringVar(&config.MongoDSN, "mongo-dsn", "", "MongoDB connection string (e.g., mongodb://localhost:27017/dbname)")
	flag.StringVar(&durationStr, "duration", "", "Duration to run load generation (e.g., 60s, 5m, 1h). If not specified, runs indefinitely")
	flag.IntVar(&config.Workers, "workers", 5, "Number of worker goroutines per database")

	flag.Parse()

	// Parse duration - if empty, set to 0 (infinite)
	if durationStr != "" {
		var err error
		config.Duration, err = time.ParseDuration(durationStr)
		if err != nil {
			fmt.Printf("Invalid duration format: %v\n", err)
			flag.Usage()
			os.Exit(1)
		}
	} else {
		config.Duration = 0 // 0 means infinite
	}

	if config.MySQLDSN == "" && config.PostgresDSN == "" && config.MongoDSN == "" {
		fmt.Println("At least one DSN must be provided")
		flag.Usage()
		os.Exit(1)
	}

	lg := &LoadGenerator{
		config:  config,
		tracker: NewProgressTracker(),
	}
	lg.ctx, lg.cancel = context.WithCancel(context.Background())

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Print("\033[2J\033[H") // Clear screen
		fmt.Println("Received shutdown signal, stopping load generation...")
		lg.cancel()
	}()

	if err := lg.Run(); err != nil {
		fmt.Print("\033[2J\033[H") // Clear screen
		log.Fatalf("Error running load generator: %v", err)
	}
}

func (lg *LoadGenerator) Run() error {
	// Show initial setup messages
	if lg.config.Duration > 0 {
		fmt.Printf("Starting load generation for %v with %d workers per database\n", lg.config.Duration, lg.config.Workers)
	} else {
		fmt.Printf("Starting load generation indefinitely with %d workers per database\n", lg.config.Workers)
	}

	// Run migrations and seed data for each configured database
	if lg.config.MySQLDSN != "" {
		fmt.Println("Running MySQL migrations...")
		mysqlService := mysql.New(lg.config.MySQLDSN, lg.config.Workers, lg.tracker)
		if err := mysqlService.RunMigrations(); err != nil {
			return fmt.Errorf("MySQL migrations failed: %w", err)
		}
		fmt.Println("MySQL migrations completed successfully")
		fmt.Println("Seeding MySQL with test data...")
		if err := mysqlService.SeedData(); err != nil {
			return fmt.Errorf("MySQL data seeding failed: %w", err)
		}
		fmt.Println("Starting MySQL load with", lg.config.Workers, "workers")
		mysqlService.StartLoad(lg.ctx, &lg.wg)
	}

	if lg.config.PostgresDSN != "" {
		fmt.Println("Running PostgreSQL migrations...")
		postgresService := postgres.New(lg.config.PostgresDSN, lg.config.Workers, lg.tracker)
		if err := postgresService.RunMigrations(); err != nil {
			return fmt.Errorf("PostgreSQL migrations failed: %w", err)
		}
		fmt.Println("PostgreSQL migrations completed successfully")
		fmt.Println("Seeding PostgreSQL with test data...")
		if err := postgresService.SeedData(); err != nil {
			return fmt.Errorf("PostgreSQL data seeding failed: %w", err)
		}
		fmt.Println("Starting PostgreSQL load with", lg.config.Workers, "workers")
		postgresService.StartLoad(lg.ctx, &lg.wg)
	}

	if lg.config.MongoDSN != "" {
		fmt.Println("Seeding MongoDB with test data...")
		mongoService := mongo.New(lg.config.MongoDSN, lg.config.Workers, lg.tracker)
		if err := mongoService.SeedData(lg.ctx); err != nil {
			return fmt.Errorf("MongoDB data seeding failed: %w", err)
		}
		fmt.Println("Starting MongoDB load with", lg.config.Workers, "workers")
		mongoService.StartLoad(lg.ctx, &lg.wg)
	}

	// Small delay to let workers start
	time.Sleep(2 * time.Second)

	// Start progress display
	go lg.tracker.DisplayProgress(lg.ctx, lg.config)

	// Wait for duration or cancellation
	if lg.config.Duration > 0 {
		// Finite duration - use timer
		timer := time.NewTimer(lg.config.Duration)
		defer timer.Stop()

		select {
		case <-timer.C:
			fmt.Print("\033[2J\033[H") // Clear screen
			fmt.Println("Load generation duration completed")
		case <-lg.ctx.Done():
			fmt.Print("\033[2J\033[H") // Clear screen
			fmt.Println("Load generation cancelled")
		}
	} else {
		// Infinite duration - wait for cancellation only
		<-lg.ctx.Done()
		fmt.Print("\033[2J\033[H") // Clear screen
		fmt.Println("Load generation stopped")
	}

	lg.cancel()
	lg.wg.Wait()

	// Show final stats
	mysqlOps, postgresOps, mongoOps, mysqlErrors, postgresErrors, mongoErrors, elapsed := lg.tracker.GetStats()
	fmt.Printf("\nFinal Results (ran for %v):\n", elapsed.Truncate(time.Second))

	if lg.config.MySQLDSN != "" {
		fmt.Printf("MySQL:      %d operations, %d errors\n", mysqlOps, mysqlErrors)
	}
	if lg.config.PostgresDSN != "" {
		fmt.Printf("PostgreSQL: %d operations, %d errors\n", postgresOps, postgresErrors)
	}
	if lg.config.MongoDSN != "" {
		fmt.Printf("MongoDB:    %d operations, %d errors\n", mongoOps, mongoErrors)
	}

	totalOps := mysqlOps + postgresOps + mongoOps
	totalErrors := mysqlErrors + postgresErrors + mongoErrors
	fmt.Printf("Total:      %d operations, %d errors\n", totalOps, totalErrors)

	fmt.Println("Load generation finished")
	return nil
}
