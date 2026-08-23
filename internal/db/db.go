package db

import (
	"fmt"
	"github.com/weeb-vip/thetvdb-enrichment/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"
)

type DB struct {
	DB *gorm.DB
}

func NewDB(cfg config.DBConfig) *DB {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DataBase, cfg.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get database connection")
	}

	// This had no pool configuration at all, which is worse than a large one:
	// Go's defaults are unlimited MaxOpenConns and MaxIdleConns of 2. So the
	// ceiling was unbounded against a database that allows 79 connections in
	// total, and only two of them were ever retained -- every connection past the
	// second was built per query, paying a TCP connect, TLS handshake and
	// Postgres auth each time against RDS over the internet.
	//
	// 2 matched open-to-idle: this is a Kafka consumer, so it handles one message
	// at a time and needs the write plus a spare, held open and reused.
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(2)

	// Long enough that connections survive quiet periods and get reused, short
	// enough that a failover or DNS change is picked up without a restart.
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return &DB{DB: db}
}
