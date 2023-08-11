package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/valyala/fasthttp"
)

func handler(queries []string, db *pgxpool.Pool) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// Get the domain query parameter.
		domain := string(ctx.QueryArgs().Peek("domain"))
		if domain == "" {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			_, _ = ctx.WriteString("domain query parameter cannot be empty")
			return
		}
		domain = strings.ToLower(domain)

		// Build the batch.
		batch := &pgx.Batch{}
		for _, query := range queries {
			batch.Queue(query, domain)
		}

		// Execute the batch.
		batchResults := db.SendBatch(ctx, batch)
		defer batchResults.Close()
		for range queries {
			// Get the result.
			var exists bool
			if err := batchResults.QueryRow().Scan(&exists); err != nil {
				// Log and return a 500.
				_, _ = fmt.Fprintln(os.Stderr, "failed to execute query:", err)
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				_, _ = ctx.WriteString("internal server error")
				return
			}

			// If the domain exists, return a 204.
			if exists {
				ctx.SetStatusCode(fasthttp.StatusNoContent)
				return
			}
		}

		// If the domain does not exist, return a 404.
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		_, _ = ctx.WriteString("domain not found")
	}
}

type oneOf struct {
	TableName string `json:"table_name"`
	Column    string `json:"column"`
}

type config struct {
	PostgresURI string  `json:"postgres_uri"`
	OneOf       []oneOf `json:"one_of"`
}

func loadConfig() *config {
	// Get the config.
	s := os.Getenv("CONFIG")
	if s == "" {
		// Read from config.json.
		b, err := os.ReadFile("config.json")
		if err != nil {
			panic(err)
		}
		s = string(b)
	} else {
		// Base64 decode the config.
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			panic(err)
		}
		s = string(b)
	}

	// Parse the config.
	var c config
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		panic(err)
	}
	return &c
}

func main() {
	// Load the config.
	conf := loadConfig()

	// Build the SQL queries to check if exists.
	queries := make([]string, len(conf.OneOf))
	for i, o := range conf.OneOf {
		// Get the table name and column name.
		tableName := o.TableName
		column := o.Column
		if tableName == "" {
			panic("table_name cannot be empty")
		}
		if column == "" {
			panic("column cannot be empty")
		}

		// Build the query.
		queries[i] = "SELECT EXISTS (SELECT 1 FROM " + tableName + " WHERE " + column + " = $1)"
	}

	// Connect to the database.
	connString := conf.PostgresURI
	if connString == "" {
		panic("postgres_uri cannot be empty")
	}
	db, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		panic(err)
	}

	// Listen on the socket.
	host := os.Getenv("HOST")
	if host == "" {
		host = ":8383"
	}
	ln, err := net.Listen("tcp", host)
	if err != nil {
		panic(err)
	}

	// Print a nice log message with a emoji.
	fmt.Println("🚀 Listening on", ln.Addr().String())

	// Serve the requests.
	if err := fasthttp.Serve(ln, handler(queries, db)); err != nil {
		panic(err)
	}
}
