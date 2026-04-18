package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/triasbrata/higo/database/rds"
	"github.com/triasbrata/higo/database/rds/driverSwitcher"
	"go.uber.org/fx"
)

type RdsFactoryOut struct {
	Driver     rds.DbDriver
	Connection rds.DbConnectionString
}
type RdsFactory func() RdsFactoryOut

func LoadRelationalDatabase(factory ...RdsFactory) fx.Option {
	options := make([]fx.Option, 0)
	switch len(factory) {
	case 1:
		options = provideConnectionMasterSlave(options, factory[0], factory[0])
	case 2:
		options = provideConnectionMasterSlave(options, factory[0], factory[1])
	default:
		options = append(options, fx.Provide(func() (*sqlx.DB, error) {
			return nil, fmt.Errorf("when call LoadRelationalDatabase the factory cant empty")
		}))
	}
	return fx.Module("pkgs/database", options...)
}

func provideConnectionMasterSlave(options []fx.Option, master, slave RdsFactory) []fx.Option {
	return append(options,
		fx.Provide(fx.Private, fx.Annotate(master,
			fx.ResultTags(`name:"driver_master"`, `name:"str_con_master"`),
		)),
		fx.Provide(fx.Annotate(driverSwitcher.OpenByDriver,
			fx.ParamTags(`name:"driver_master"`, `name:"str_con_master"`),
			fx.ResultTags(`name:con_master`),
		)),
		fx.Provide(fx.Private, fx.Annotate(slave,
			fx.ResultTags(`name:"driver_slave"`, `name:"str_con_slave"`),
		)),
		fx.Provide(fx.Annotate(driverSwitcher.OpenByDriver,
			fx.ParamTags(`name:"driver_slave"`, `name:"str_con_slave"`),
			fx.ResultTags(`name:con_slave`),
		)))
}
