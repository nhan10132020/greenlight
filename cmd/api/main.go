package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
}

type application struct {
	config config
	logger *log.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", "host=localhost user=root password=123456Aa dbname=greenlight port=5432", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	_, postgresDB, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}
	defer postgresDB.Close()

	logger.Printf("database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Printf("Starting %s server on %s", cfg.env, srv.Addr)
	err = srv.ListenAndServe()
	logger.Fatal(err)
}

func openDB(cfg config) (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.db.dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	postgresDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	postgresDB.SetMaxOpenConns(cfg.db.maxOpenConns)
	postgresDB.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, nil, err
	}
	postgresDB.SetConnMaxIdleTime(duration)

	// create a context with a 5-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use PingContext() to establish a new connection to the database, passing in the
	// context as a parameter. If the connection couldn't be
	// established successfully within the 5 second deadline, then this will return an
	// error
	err = postgresDB.PingContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	return db, postgresDB, nil
}
