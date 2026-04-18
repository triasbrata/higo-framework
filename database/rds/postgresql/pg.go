package postgresql

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/triasbrata/higo/database/rds"
	"github.com/triasbrata/higo/database/rds/utils"
)

type pgdb struct {
	dbSlave  *sqlx.DB
	dbMaster *sqlx.DB
}

// Exec implements rds.RelationalDatabase.
func (p *pgdb) Exec(ctx context.Context, uniqueKey string, query string, queryParam interface{}) (sql.Result, error) {
	var db sqlx.ExtContext
	db = p.dbSlave
	tx, ok := rds.InTransaction.Get(ctx)
	if ok {
		db = tx
	}
	res, err := sqlx.NamedExecContext(ctx, db, query, queryParam)
	if err != nil {
		return rds.NoRow{}, err
	}
	return res, err
}

// QueryOne implements rds.RelationalDatabase.
func (p *pgdb) Query(ctx context.Context, uniqueKey string, query string, queryParam interface{}) (*sqlx.Rows, error) {
	var db sqlx.ExtContext
	db = p.dbSlave
	tx, ok := rds.InTransaction.Get(ctx)
	if ok {
		db = tx
	}
	res, err := sqlx.NamedQueryContext(ctx, db, query, queryParam)
	if err != nil {
		return &sqlx.Rows{}, err
	}
	return res, err
}

// Transaction implements rds.TransactionDatabase.
func (p *pgdb) Transaction(ctx context.Context, trOption *sql.TxOptions, wfx func(ctx context.Context) error) (err error) {
	tx, err := p.dbMaster.BeginTxx(ctx, trOption)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = wfx(rds.InTransaction.Set(ctx, tx))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func OpenDBPostgres(param rds.ParamOpenCon) (*sqlx.DB, error) {
	return utils.OpenDB(param)
}

func PostgresqlTx() rds.TransactionDatabase {
	return &pgdb{}
}
func Postgresql() rds.RelationalDatabase {
	return &pgdb{}
}
