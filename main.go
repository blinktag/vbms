package main

import (
	"database/sql"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/blinktag/vbms/server"
	"github.com/caarlos0/env"
	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	UpdateTick int `env:"UPDATE_TICK" envDefault:"5"`
	BatchSize  int `env:"BATCH_SIZE" envDefault:"10"`
}

// cfg holds the application configuration
var cfg config

// Servers holds all servers we wish to monitor
var Servers []*server.Server

func main() {

	loadEnvironment()
	verifyDatabase()
	runBatch() // Fire off first batch

	for range doTicker() {
		runBatch()
	}
}

// Load environment variables
func loadEnvironment() {
	env.Parse(&cfg)
}

// doTicker creates a ticker based on the UPDATE_TICK envar
func doTicker() <-chan time.Time {
	ticker := time.NewTicker(time.Second * time.Duration(cfg.UpdateTick))
	return ticker.C
}

// verifyDatabase checks that our sqlite db exists
func verifyDatabase() {
	// The sqlite3 library creates an empty file if it does not exist
	// This is not expected behavior, so check for db first using "stat"
	if _, err := os.Stat("./servers.db"); err != nil {
		log.Fatal("Unable to locate servers.db sqlite database")
		os.Exit(1)
	}
}

// loadDatabase opens sqlite3 database
func loadDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", "./servers.db")

	if err != nil {
		log.Fatal("Unable to open servers.db sqlite database")
		os.Exit(1)
	}

	return db
}

// runBatch initiates checks on a batch of servers
func runBatch() {
	db := loadDatabase()
	batchID := updateBatch(db)

	rows, err := db.Query("SELECT * FROM servers WHERE lastupdate = ?", batchID)

	if err != nil {
		log.Fatal("Unable to select rows from database")
	}

	// Ensure cleanup
	defer rows.Close()

	for rows.Next() {

		srv := server.NewServer(db, rows)

		go func(cur *server.Server) {
			cur.RunChecks()
		}(&srv)
	}
}

// updateBatch updates a chunk of server rows with a lock value
func updateBatch(db *sql.DB) int64 {

	// Current timestamp will be used as a batch lock
	now := time.Now().Unix()

	// No checks quicker than 60 seconds. Don't want to DOS ourselves
	limit := now - 60

	// Update batch of servers
	// sqlite doesn't like LIMIT clauses in UPDATE statements, so do a hacky subquery
	stmt, err := db.Prepare(`
		UPDATE servers SET lastupdate = ?
		WHERE id IN (SELECT id FROM servers WHERE lastupdate < ? LIMIT ?)
	`)

	if err != nil {
		log.Fatal(err)
	}

	res, err := stmt.Exec(now, limit, cfg.BatchSize)

	if err != nil {
		log.Fatal(err)
	}

	rows, _ := res.RowsAffected()
	log.Infof("Batch of %d servers queued for updates", rows)

	// Return current batch ID
	return now
}
