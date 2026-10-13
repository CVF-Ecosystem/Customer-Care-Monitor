package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "synthetic SQL DSN missing")
		os.Exit(2)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open synthetic SQL connection: %T\n", err)
		os.Exit(1)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "host loopback app-user PingContext failed: %T\n", err)
		os.Exit(1)
	}
	var value int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&value); err != nil || value != 1 {
		fmt.Fprintln(os.Stderr, "host loopback app-user SELECT 1 failed")
		os.Exit(1)
	}
	fmt.Println("host loopback authenticated app-user PingContext=PASS; SELECT 1=PASS")
}
