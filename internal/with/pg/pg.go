//go:build with_pg

package pg

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/machbase/neo-shell/internal/with"
	spi "github.com/machbase/neo-spi"
)

func New(name string, addr string) spi.FactoryFunc {
	d := &drv{
		driver: "postgres",
		addr:   "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable",
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
