package db

import "database/sql"

// Store provides all functions to execute db queries and transactions
type Store struct {
	*Queries
	db *sql.DB
}