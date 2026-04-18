package rds

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/triasbrata/higo/contextw"
	"go.uber.org/fx"
)

type DbConnectionString string
type DbDriver string
type ParamOpenCon struct {
	Driver DbDriver
	Url    DbConnectionString
	Lc     fx.Lifecycle
}

const (
	InTransaction contextw.ContextValue[*sqlx.Tx] = "__db_tx__"
)

type NoRow struct {
}

// LastInsertId implements sql.Result.
func (n NoRow) LastInsertId() (int64, error) {
	return 0, nil
}

// RowsAffected implements sql.Result.
func (n NoRow) RowsAffected() (int64, error) {
	return 0, nil
}

var _ sql.Result = NoRow{}

type RelationalDatabase interface {
	Exec(ctx context.Context, uniqueKey string, query string, queryParam interface{}) (sql.Result, error)
	Query(ctx context.Context, uniqueKey string, query string, queryParam interface{}) (*sqlx.Rows, error)
}

type TransactionDatabase interface {
	Transaction(ctx context.Context, trOption *sql.TxOptions, wfx func(ctx context.Context) error) error
}
type DbxTx interface {
}
