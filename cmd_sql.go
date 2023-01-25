package shell

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/machrpc"
	"golang.org/x/term"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "sql",
		Aliases: []string{`\s`},
		PcFunc:  pcSql,
		Action:  doSql,
		Desc:    "run sql query",
		Usage:   "  sql <sql query>",
	})
}

func pcSql(cc Client) readline.PrefixCompleterInterface {
	cli := cc.(*client)
	return readline.PcItem("sql",
		readline.PcItemDynamic(cli.SqlHistory),
	)
}

func doSql(cc Client, sqlText string) {
	cli := cc.(*client)
	rows, err := cli.db.Query(sqlText)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	defer rows.Close()

	cli.AddSqlHistory(sqlText)

	if !rows.IsFetchable() {
		cli.Println(rows.Message())
		return
	}

	cols, err := rows.Columns()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	names := make([]any, len(cols)+1)
	names[0] = "#"
	for i := range cols {
		if cols[i].Type == "datetime" {
			names[i+1] = fmt.Sprintf("%s(%s)", cols[i].Name, cli.conf.TimeLocation.String())
		} else {
			names[i+1] = cols[i].Name
		}
	}

	rec := makeBuffer(cols)

	box := cli.NewBox(names, !cli.Interactive())

	windowHeight := 0
	//windowWidth := 0
	if term.IsTerminal(0) {
		if _, height, err := term.GetSize(0); err == nil {
			windowHeight = height
			//windowWidth = width
		}
	}

	height := windowHeight - 4
	if cli.conf.Heading {
		height--
	}

	nrow := 0
	for {
		if !rows.Next() {
			box.Render()
			box.ResetRows()
			return
		}
		err := rows.Scan(rec...)
		if err != nil {
			cli.Println("ERR>", err.Error())
			return
		}
		nrow++
		vs := makeValues(rec, cli.conf.TimeLocation)
		values := make([]any, len(vs)+1)
		values[0] = nrow
		for i := range vs {
			values[i+1] = vs[i]
		}
		box.AppendRow(values...)

		if windowHeight > 0 && nrow%height == 0 {
			box.Render()
			box.ResetRows()
			if cli.interactive {
				cli.Print(":")
				// switch stdin into 'raw' mode
				if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
					b := make([]byte, 3)
					if _, err = os.Stdin.Read(b); err == nil {
						term.Restore(int(os.Stdin.Fd()), oldState)
						switch b[0] {
						case 'q', 'Q':
							return
						default:
						}
						// ':' prompt를 삭제한다.
						// erase line
						fmt.Fprintf(os.Stdout, "%s%s", "\x1b", "[2K")
						// cursor backward
						fmt.Fprintf(os.Stdout, "%s%s", "\x1b", "[1D")
					}
				}
			} else {
				box.ResetHeaders()
			}
		}
	}
}

func makeValues(rec []any, tz *time.Location) []string {
	cols := make([]string, len(rec))
	for i, r := range rec {
		if r == nil {
			cols[i] = "NULL"
			continue
		}
		switch v := r.(type) {
		case *string:
			cols[i] = *v
		case *time.Time:
			timeformat := "2006-01-02 15:04:05.000000"
			cols[i] = v.In(tz).Format(timeformat)
		case *float64:
			cols[i] = fmt.Sprintf("%f", *v)
		case *int:
			cols[i] = fmt.Sprintf("%d", *v)
		case *int32:
			cols[i] = fmt.Sprintf("%d", *v)
		case *int64:
			cols[i] = fmt.Sprintf("%d", *v)
		default:
			cols[i] = fmt.Sprintf("%T", r)
		}
	}
	return cols
}

func makeBuffer(cols []*machrpc.Column) []any {
	rec := make([]any, len(cols))
	for i := range cols {
		switch cols[i].Type {
		case "int16":
			rec[i] = new(int16)
		case "int32":
			rec[i] = new(int32)
		case "int64":
			rec[i] = new(int64)
		case "datetime":
			rec[i] = new(time.Time)
		case "float":
			rec[i] = new(float32)
		case "double":
			rec[i] = new(float64)
		case "ipv4":
			rec[i] = new(net.IP)
		case "ipv6":
			rec[i] = new(net.IP)
		case "string":
			rec[i] = new(string)
		case "binary":
			rec[i] = new([]byte)
		}
	}
	return rec
}
