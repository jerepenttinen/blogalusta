package main

import (
	"blogalusta/internal/data"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golangcollege/sessions"
	"github.com/gomarkdown/markdown/html"
	"github.com/microcosm-cc/bluemonday"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var (
	buildTime string
	version   string
)

type contextKey string

var (
	contextKeyUser        = contextKey("user")
	contextKeyPublication = contextKey("publication")
	contextKeyWriters     = contextKey("writers")
	contextKeyPending     = contextKey("pending")
	contextKeyArticle     = contextKey("article")
	contextKeyProfile     = contextKey("profileUser")
	contextKeyComment     = contextKey("comment")
)

type config struct {
	port    int
	useHsts bool

	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}

	avatar struct {
		maxSize    int
		sideLength int
	}
}

type application struct {
	errorLog      *log.Logger
	infoLog       *log.Logger
	config        config
	models        data.Models
	session       *sessions.Session
	templateCache map[string]*template.Template
	markdown      struct {
		policy   *bluemonday.Policy
		renderer *html.Renderer
	}
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", getEnvInt("PORT", 4000), "API server port")
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DATABASE_URL"), "PostgreSQL DSN")
	secret := flag.String("secret", os.Getenv("SESSION_SECRET"), "Session secret key")
	flag.BoolVar(&cfg.useHsts, "hsts", getEnvBool("USE_HSTS", false), "Upgrade to https automatically")

	// Heroku free DB has max 20 connections
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 20, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 20, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.IntVar(&cfg.avatar.maxSize, "avatar-max-size", 1024*1024, "Avatar max size")
	flag.IntVar(&cfg.avatar.sideLength, "avatar-side-length", 256, "Avatar size length")

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Buildtime:\t%s\n", buildTime)
		os.Exit(0)
	}

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	templateCache, err := newTemplateCache("./ui/html")
	if err != nil {
		log.Fatal(err)
	}

	db, err := openDB(cfg)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	session := sessions.New([]byte(*secret))
	session.Lifetime = 24 * time.Hour
	session.Secure = true
	session.SameSite = http.SameSiteStrictMode

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}

	renderer := html.NewRenderer(opts)

	app := &application{
		config:        cfg,
		infoLog:       infoLog,
		errorLog:      errorLog,
		models:        data.NewModels(db),
		templateCache: templateCache,
		session:       session,
		markdown: struct {
			policy   *bluemonday.Policy
			renderer *html.Renderer
		}{
			policy:   bluemonday.UGCPolicy(),
			renderer: renderer,
		},
	}

	infoLog.Printf("starting server on port %d\n", app.config.port)
	if app.config.useHsts {
		infoLog.Println("using hsts")
	}
	err = app.serve()
	if err != nil {
		errorLog.Fatal(err)
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

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return nil, err
	}

	m.Up()

	return db, nil
}
