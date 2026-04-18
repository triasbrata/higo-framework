package driverSwitcher

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/triasbrata/higo/database/rds"
	"github.com/triasbrata/higo/database/rds/postgresql"
)

func OpenByDriver(param rds.ParamOpenCon) (*sqlx.DB, error) {
	switch param.Driver {
	case "postgres":
		return postgresql.OpenDBPostgres(param)
	default:
		return nil, fmt.Errorf("driver %s not support", param.Driver)
	}
}
