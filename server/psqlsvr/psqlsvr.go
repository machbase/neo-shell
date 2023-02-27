//go:build neo_shell

package psqlsvr

import (
	"context"
	"log"

	wire "github.com/jeroenrinzema/psql-wire"
	"github.com/lib/pq/oid"
	logging "github.com/machbase/neo-logging"
	spi "github.com/machbase/neo-spi"
)

func Enabled() bool {
	return true
}

type Config struct {
	Address string
}

type Server interface {
	Start() error
	Stop()
}

func New(db spi.Database, conf *Config) (Server, error) {
	return &svr{
		conf: conf,
		log:  logging.GetLog("psqlsvr"),
	}, nil
}

type svr struct {
	conf       *Config
	log        logging.Log
	underlying *wire.Server
}

func (s *svr) Start() error {
	options := []wire.OptionFn{
		wire.SimpleQuery(s.handle),
	}
	ns, err := wire.NewServer(options...)
	if err != nil {
		return err
	}
	s.underlying = ns
	s.underlying.ListenAndServe(s.conf.Address)
	return nil
}

func (s *svr) Stop() {
}

var table = wire.Columns{
	{
		Table:  0,
		Name:   "name",
		Oid:    oid.T_text,
		Width:  256,
		Format: wire.TextFormat,
	},
	{
		Table:  0,
		Name:   "member",
		Oid:    oid.T_bool,
		Width:  1,
		Format: wire.TextFormat,
	},
	{
		Table:  0,
		Name:   "age",
		Oid:    oid.T_int4,
		Width:  1,
		Format: wire.TextFormat,
	},
}

func (s *svr) handle(ctx context.Context, query string, writer wire.DataWriter, parameters []string) error {
	log.Println("incoming SQL query:", query)

	if query == `SHOW server_version` {
		writer.Define(wire.Columns{
			{
				Table:  0,
				Name:   "server_version",
				Oid:    oid.T_text,
				Width:  256,
				Format: wire.TextFormat,
			}})
		writer.Row([]any{"10.12"})
	} else if query == `SELECT datname "Database" FROM pg_database WHERE datistemplate = false order by datname ASC;` {
		writer.Define(wire.Columns{
			{
				Table:  0,
				Name:   "Database",
				Oid:    oid.T_text,
				Width:  256,
				Format: wire.TextFormat,
			}})
		writer.Row([]any{"machbase"})
	} else {
		writer.Define(table)
		writer.Row([]any{"John", true, 29})
		writer.Row([]any{"Marry", false, 21})
	}
	return writer.Complete("OK")
}
