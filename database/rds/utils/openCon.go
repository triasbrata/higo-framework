package utils

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/triasbrata/higo/database/rds"
	"go.uber.org/fx"
)

func OpenDB(param rds.ParamOpenCon) (*sqlx.DB, error) {
	db, err := sqlx.Open(string(param.Driver), string(param.Url))
	if err != nil {
		return nil, err
	}
	param.Lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return db.PingContext(ctx)
	}, OnStop: func(ctx context.Context) error {
		return db.Close()
	}})

	return db, nil
}
