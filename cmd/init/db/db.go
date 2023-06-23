package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// maxIdleConns = 10
	maxOpenConns = 100
)

func InitPostgresDB() (*pgxpool.Pool, error) {

	dbInfo := fmt.Sprintf(
		"host=%s dbname=%s user=%s password=%s pool_max_conns=%d",

		"localhost",
		"db_forum",
		"db_forum",
		"db_forum",
		maxOpenConns,
	)

	return pgxpool.New(context.Background(), dbInfo)
}
