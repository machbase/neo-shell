package wiresvr

import (
	"context"
	"net"
	"strconv"
	"strings"

	wire "github.com/jeroenrinzema/psql-wire"
	"github.com/jeroenrinzema/psql-wire/codes"
	pgerr "github.com/jeroenrinzema/psql-wire/errors"
	"github.com/lib/pq/oid"
	logging "github.com/machbase/neo-logging"
	spi "github.com/machbase/neo-spi"
	"go.uber.org/zap"
)

type Config struct {
	Listeners   []string
	Development bool
}

type Server interface {
	Start() error
	Stop()
}

func New(db spi.Database, conf *Config) (Server, error) {
	return &svr{
		log:  logging.GetLog("wiresvr"),
		conf: conf,
		db:   db,
	}, nil
}

type svr struct {
	log   logging.Log
	conf  *Config
	db    spi.Database
	lsnrs []*wire.Server
}

func (s *svr) Start() (err error) {
	options := []wire.OptionFn{}
	options = append(options, wire.Parse(s.parse))
	options = append(options, wire.Version("9.0"))
	if s.conf.Development {
		zlog, _ := zap.NewDevelopment()
		options = append(options, wire.Logger(zlog))
	} else {
		zapCfg := zap.NewProductionConfig()
		zapCfg.Level.SetLevel(zap.ErrorLevel)
		zlog, _ := zapCfg.Build()
		options = append(options, wire.Logger(zlog))
	}
	for _, addr := range s.conf.Listeners {
		lsnr, err := wire.NewServer(options...)
		if err != nil {
			return err
		}
		tcpaddr := strings.TrimPrefix(addr, "tcp://")
		l, err := net.Listen("tcp", tcpaddr)
		if err != nil {
			return err
		}
		go lsnr.Serve(l)
		s.log.Infof("WIRE Listen %s", addr)
	}
	return nil
}

func (s *svr) Stop() {
	for _, l := range s.lsnrs {
		l.Close()
	}
}

func (s *svr) parse(ctx context.Context, query string) (wire.PreparedStatementFn, []oid.Oid, error) {
	// NOTE: we have to lookup all parameters within the given query.
	// Parameters could represent positional parameters or anonymous
	// parameters. We return a zero parameter oid for each parameter
	// indicating that the given parameters could contain any type. We
	// could safely ignore the err check while converting given
	// parameters since ony matches are returned by the positional
	// parameter regex.
	matches := wire.QueryParameters.FindAllStringSubmatch(query, -1)
	parameters := make([]oid.Oid, 0, len(matches))

	for _, match := range matches {
		// NOTE: we have to check whether the returned match is a
		// positional parameter or an un-positional parameter.
		// SELECT * FROM users WHERE id = ?
		if match[1] == "" {
			parameters = append(parameters, 0)
		}

		position, _ := strconv.Atoi(match[1]) //nolint:errcheck
		if position > len(parameters) {
			parameters = parameters[:position]
		}
	}

	var statement wire.PreparedStatementFn

	if strings.HasPrefix(strings.ToUpper(query), "SET ") {
		statement = func(ctx context.Context, writer wire.DataWriter, parameters []string) error {
			return s.handleSet(ctx, query, writer, parameters)
		}
	} else {
		statement = func(ctx context.Context, writer wire.DataWriter, parameters []string) error {
			return s.handleQuery(ctx, query, writer, parameters)
		}
	}

	return statement, parameters, nil
}

func (s *svr) handleSet(ctx context.Context, query string, writer wire.DataWriter, parameters []string) error {
	s.log.Debug("handle set", query)
	//writer.Define(wire.Columns{
	// wire.Column{
	// 	Table:  int32(0),
	// 	Name:   "extra_float_digits",
	// 	Oid:    oid.T_text,
	// 	Width:  int16(1),
	// 	Format: wire.TextFormat,
	// },
	//})
	//writer.Row([]any{})
	return writer.Complete("SET")
}

func (s *svr) handleQuery(ctx context.Context, query string, writer wire.DataWriter, parameters []string) error {
	s.log.Debug("handle query", query)

	params := make([]any, len(parameters))
	for i, p := range parameters {
		params[i] = p
	}

	rows, err := s.db.Query(query, params...)
	if err != nil {
		s.log.Error(err.Error())
		err = pgerr.WithCode(err, codes.Internal)
		err = pgerr.WithSeverity(err, pgerr.LevelFatal)
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		s.log.Error(err.Error())
		return err
	}

	tableId := 0
	define := wire.Columns{}

	for _, c := range cols {
		oidType, format := columnToOid(c)
		def := wire.Column{
			Table:  int32(tableId),
			Name:   c.Name,
			Oid:    oidType,
			Width:  int16(c.Size),
			Format: format,
		}
		define = append(define, def)
	}
	writer.Define(define)

	values := cols.MakeBuffer()
	for rows.Next() {
		rows.Scan(values...)
		writer.Row(values)
	}

	// err := errors.New("unimplemented feature")
	// err = psqlerr.WithCode(err, codes.FeatureNotSupported)
	// err = psqlerr.WithSeverity(err, psqlerr.LevelFatal)

	return writer.Complete("OK")
}

func columnToOid(c *spi.Column) (oid.Oid, wire.FormatCode) {
	oidType := oid.T_text
	format := wire.TextFormat
	switch c.Type {
	case "int16":
		oidType = oid.T_int2
	case "int32":
		oidType = oid.T_int4
	case "int64":
		oidType = oid.T_int8
	case "datetime":
		oidType = oid.T_timestamp
	case "float":
		oidType = oid.T_float4
	case "double":
		oidType = oid.T_float8
	case "ipv4":
		oidType = oid.T_inet
		format = wire.BinaryFormat
	case "ipv6":
		oidType = oid.T_inet
		format = wire.BinaryFormat
	case "string":
		oidType = oid.T_text
	case "binary":
		oidType = oid.T_bytea
		format = wire.BinaryFormat
	}
	return oidType, format
}
