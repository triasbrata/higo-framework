package database

import "github.com/triasbrata/higo-framework/database/rds"

type RdsFactoryOut struct {
	Driver     rds.DbDriver
	Connection rds.DbConnectionString
}

type RdsFactory func() RdsFactoryOut
