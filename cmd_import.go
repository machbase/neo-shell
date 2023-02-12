package shell

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/spi"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "import",
		PcFunc: pcImport,
		Action: doImport,
		Desc:   "import table",
		Usage:  helpImport,
	})
}

const helpImport = `  import [options] <table>
    table               table name to write
  options:
    --input,-i <file>  input file, (default: '-' stdin)
    --format,-f        file format [csv] (default:'csv')
    --no-header        there is no header, do not skip first line (default)
    --header           first line is header, skip it
    --method           write method [insert|append] (default:'insert')
    --delimiter,-d     csv delimiter (default:',')
    --tz               timezone for handling datetime
    --timeformat,-t    time format [ns|ms|s|<timeformat>] (default:'ns')
       ns, us, ms, s
         represents unix epoch time in nano-, micro-, milli- and seconds for each
       timeformat
         consult "help timeformat"
    --eof <string>     specify eof line, use any string matches [a-zA-Z0-9]+ (default: '.')`

type ImportCmd struct {
	Table        string         `arg:"" name:"table"`
	Input        string         `name:"input" short:"i" default:"-"`
	HasHeader    bool           `name:"header" negatable:""`
	EofMark      string         `name:"eof" default:"."`
	InputFormat  string         `name:"format" short:"f" default:"csv"`
	Method       string         `name:"method" default:"insert"`
	Delimiter    string         `name:"delimiter" short:"d" default:","`
	TimeFormat   string         `name:"timeformat" short:"t" default:"ns"`
	TimeLocation *time.Location `name:"tz" default:"UTC"`
	Help         bool           `kong:"-"`
}

func pcImport(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("import")
}

func doImport(cli Client, cmdLine string) {
	cmd := &ImportCmd{}
	parser, err := Kong(cmd, func() error { cli.Println(helpImport); cmd.Help = true; return nil })
	if err != nil {
		cli.Println(err.Error())
		return
	}

	_, err = parser.Parse(splitFields(cmdLine, true))
	if cmd.Help {
		return
	}
	if err != nil {
		cli.Println(err.Error())
		return
	}

	var r *bufio.Reader
	if cmd.Input == "-" {
		r = bufio.NewReader(cli.Stdin())
	} else {
		f, err := os.Open(cmd.Input)
		if err != nil {
			cli.Println(err.Error())
			return
		}
		defer f.Close()
		r = bufio.NewReader(f)
	}

	db := cli.Database()
	_desc, err := spi.DoDescribe(db, cmd.Table, false)
	if err != nil {
		cli.Printfln("ERR fail to get table info '%s', %s", cmd.Table, err.Error())
		return
	}
	desc := (_desc).(*spi.TableDescription)

	if cli.Interactive() {
		cli.Printfln("# Enter %s⏎ to quit", cmd.EofMark)
		colNames := []string{}
		for _, col := range desc.Columns {
			colNames = append(colNames, col.Name)
		}

		cli.Println("#", strings.Join(colNames, cmd.Delimiter))
	}
	buff := []string{}
	vals := []any{}
	hold := []string{}
	lineno := 0
	written := 0
	for {
		bs, ispart, err := r.ReadLine()
		if err != nil {
			break
		}
		str := string(bs)
		if str == cmd.EofMark {
			break
		}
		buff = append(buff, str)

		if ispart {
			continue
		}

		lineno++
		line := strings.Join(buff, "")
		toks := strings.Split(line, cmd.Delimiter)
		if len(toks) != len(desc.Columns) {
			cli.Printfln("line %d contains %d columns, but expected %d", lineno, len(toks), len(desc.Columns))
			break
		}

		for i := 0; i < len(desc.Columns); i++ {
			str := strings.TrimSpace(toks[i])
			v, err := stringToColumnValue(str, desc.Columns[i], cmd.TimeLocation, cmd.TimeFormat)
			if err != nil {
				cli.Printfln("line %d, column %s, %s", lineno, desc.Columns[i].Name, err.Error())
				break
			}
			vals = append(vals, v)
			hold = append(hold, "?")
		}
		query := fmt.Sprintf("insert into %s values(%s)", cmd.Table, strings.Join(hold, ","))
		if err := db.Exec(query, vals...); err != nil {
			cli.Println(err.Error())
			break
		}
		written++

		buff = buff[:0]
		vals = vals[:0]
		hold = hold[:0]
	}
	cli.Println("total", written, "record(s) imported")
}

func stringToColumnValue(str string, cd *spi.ColumnDescription, tz *time.Location, timeformat string) (any, error) {
	switch cd.Type {
	case spi.Int16ColumnType:
		return strconv.ParseInt(str, 10, 16)
	case spi.Uint16ColumnType:
		return strconv.ParseUint(str, 10, 16)
	case spi.Int32ColumnType:
		return strconv.ParseInt(str, 10, 32)
	case spi.Uint32ColumnType:
		return strconv.ParseUint(str, 10, 32)
	case spi.Int64ColumnType:
		return strconv.ParseInt(str, 10, 64)
	case spi.Uint64ColumnType:
		return strconv.ParseUint(str, 10, 64)
	case spi.Float32ColumnType:
		return strconv.ParseFloat(str, 32)
	case spi.Float64ColumnType:
		return strconv.ParseFloat(str, 64)
	case spi.VarcharColumnType:
		return str, nil
	case spi.TextColumnType:
		return str, nil
	case spi.ClobColumnType:
		return str, nil
	case spi.BlobColumnType:
		return str, nil
	case spi.BinaryColumnType:
		return str, nil
	case spi.DatetimeColumnType:
		switch timeformat {
		case "ns":
			v, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return nil, err
			}
			return time.Unix(0, v), nil
		case "ms":
			v, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return nil, err
			}
			return time.Unix(0, v*int64(time.Millisecond)), nil
		case "us":
			v, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return nil, err
			}
			return time.Unix(0, v*int64(time.Microsecond)), nil
		case "s":
			v, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return nil, err
			}
			return time.Unix(v, 0), nil
		default:
			return time.ParseInLocation(timeformat, str, tz)
		}
	case spi.IpV4ColumnType:
		if ip := net.ParseIP(str); ip != nil {
			return ip, nil
		} else {
			return nil, fmt.Errorf("unable to parse as ip address %s", str)
		}
	case spi.IpV6ColumnType:
		if ip := net.ParseIP(str); ip != nil {
			return ip, nil
		} else {
			return nil, fmt.Errorf("unable to parse as ip address %s", str)
		}
	default:
		return nil, fmt.Errorf("unknown column type %d", cd.Type)
	}
}
