package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var (
	buildTime string
	version   string
)

type config struct {
	port int

	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
}

type application struct {
	config        config
	templateCache map[string]*template.Template
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", getEnvInt("PORT", 4000), "API server port")
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DATABASE_URL"), "PostgreSQL DSN")

	// Heroku free DB has max 20 connections
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 20, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 20, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()
	fmt.Printf("%s\n", cfg.db.dsn)

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Buildtime:\t%s\n", buildTime)
		os.Exit(0)
	}

	templateCache, err := newTemplateCache("./ui/html")
	if err != nil {
		log.Fatal(err)
	}

	db, err := openDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := &application{
		config:        cfg,
		templateCache: templateCache,
	}

	log.Printf("starting server on port %d\n", app.config.port)
	err = app.serve()
	if err != nil {
		log.Fatal(err)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}