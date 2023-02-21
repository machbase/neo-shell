//go:build with_mysql

package mysql

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/machbase/neo-shell/internal/with"
	spi "github.com/machbase/neo-spi"
)

func New(name string, addr string) spi.FactoryFunc {
	d := &drv{
		driver: "mysql",
		addr:   "user:password@/dbname",
	}
	return d.New
}

type drv struct {
	driver string
	addr   string
}

func (d *drv) New() (spi.Database, error) {
	db, err := sql.Open(d.driver, d.addr)
	if err != nil {
		return nil, err
	}
	return with.New(db), nil
}
