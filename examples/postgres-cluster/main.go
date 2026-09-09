// postgres-cluster example: reads config.yaml and connects to PostgreSQL
// via dbkit.Open.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	dbkit "github.com/Hallucination-Team/go-dbkit"
	"gopkg.in/yaml.v3"
)

func main() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	var cfg dbkit.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	database, err := dbkit.Open(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}

	var version string
	if err := database.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		log.Fatalf("query: %v", err)
	}
	fmt.Println("connected:", version)
}
