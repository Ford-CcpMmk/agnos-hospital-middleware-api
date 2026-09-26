package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DBTX interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}
