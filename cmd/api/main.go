package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
)

var (
	buildTime string
	version   string
)

type config struct {
	port int
}

type application struct {
	config        config
	templateCache map[string]*template.Template
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", getEnvInt("PORT", 4000), "API server port")

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Buildtime:\t%s\n", buildTime)
		os.Exit(0)
	}

	templateCache, err := newTemplateCache("./ui/html")
	if err != nil {
		log.Fatal(err)
	}

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
