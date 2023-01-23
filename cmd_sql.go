package shell

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/machbase/neo-grpc/machrpc"
	"golang.org/x/term"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "sql",
		Aliases: []string{"\\s"},
		PcFunc:  nil,
		Action:  doSql,
		Desc:    "run sql query",
		Usage:   "sql <sql query>",
	})
}

func doSql(cc Client, sqlText string, interactive bool) {
	cli := cc.(*client)
	rows, err := cli.db.Query(sqlText)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	names := make([]string, len(cols))
	for i := range cols {
		names[i] = cols[i].Name
	}
	rec := makeBuffer(cols)

	chunk := &ResultChunk{}
	chunk.heading = cli.conf.Heading
	chunk.cols = names

	if term.IsTerminal(0) {
		if width, height, err := term.GetSize(0); err == nil {
			chunk.windowHeight = height
			chunk.windowWidth = width
		}
	}

	height := chunk.windowHeight - 1
	if chunk.heading {
		height--
	}

	nrow := 0
	for {
		if !rows.Next() {
			if len(chunk.rows) > 0 {
				cli.display(chunk, interactive)
			}
			cli.Printfln(cli.Printer().Sprintf("%d rows selected", nrow))
			return
		}
		err := rows.Scan(rec...)
		if err != nil {
			cli.Println("ERR>", err.Error())
			return
		}
		nrow++
		chunk.rows = append(chunk.rows, makeValues(rec, cli.conf.LocalTime))

		if chunk.windowHeight > 0 && nrow%height == 0 {
			chunk = cli.display(chunk, interactive)
			if interactive {
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
			}
		}
	}
}

func makeValues(rec []any, localtime bool) []string {
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
			if localtime {
				cols[i] = v.Local().Format(timeformat)
			} else {
				cols[i] = v.UTC().Format(timeformat)
			}
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

type ResultChunk struct {
	heading bool
	width   []int
	cols    []string
	rows    [][]string

	windowWidth  int
	windowHeight int
}

func (cli *client) display(chunk *ResultChunk, interactive bool) *ResultChunk {
	if cli.conf.Format == Formats.CSV {
		chunk.displayCSV(cli, interactive)
	} else {
		chunk.displayDefault(cli, interactive)
	}
	return &ResultChunk{
		heading:      chunk.heading,
		width:        chunk.width,
		cols:         chunk.cols,
		windowWidth:  chunk.windowWidth,
		windowHeight: chunk.windowHeight,
	}
}

func (chunk *ResultChunk) displayDefault(cli *client, interactive bool) {
	if len(chunk.width) == 0 {
		chunk.width = make([]int, len(chunk.cols))
		// 각 컬럼의 폭을 계산한다.
		for c := range chunk.cols {
			// 컬럼 명의 길이를 최소 폭으로 한다.
			max := len(chunk.cols[c])
			// 각 rows를 순회하며 해당 column 값의 폭 중에서 가장 긴 값을 찾는다.
			for r := range chunk.rows {
				v := chunk.rows[r][c]
				if len(v) > max {
					max = len(v)
				}
			}
			chunk.width[c] = max
		}
		for c := range chunk.cols {
			chunk.cols[c] = fmt.Sprintf("%-*s", chunk.width[c], chunk.cols[c])
		}
	}

	if chunk.heading && interactive {
		line := strings.Join(chunk.cols, " | ")
		if chunk.windowWidth > 0 && len(line) > chunk.windowWidth {
			line = line[0 : chunk.windowWidth-4]
			line = line + "..."
		}
		cli.Println(line)
	}
	for r, row := range chunk.rows {
		for c := range chunk.cols {
			chunk.rows[r][c] = fmt.Sprintf("%-*s", chunk.width[c], row[c])
		}
		line := strings.Join(row, "   ")
		if chunk.windowWidth > 0 && len(line) > chunk.windowWidth {
			line = line[0 : chunk.windowWidth-4]
			line = line + "..."
		}
		cli.Println(line)
	}
}

func (chunk *ResultChunk) displayCSV(cli *client, interactive bool) {
	if len(chunk.width) == 0 {
		chunk.width = make([]int, len(chunk.cols))
		// CSV header 출력
		cli.Println(strings.Join(chunk.cols, ","))
	}

	for r, row := range chunk.rows {
		var line string
		if cli.conf.Format == Formats.CSV {
			line = strings.Join(row, ",")
		} else {
			for c := range chunk.cols {
				chunk.rows[r][c] = fmt.Sprintf("%-*s", chunk.width[c], row[c])
			}
			line = strings.Join(row, "   ")
			if chunk.windowWidth > 0 && len(line) > chunk.windowWidth {
				line = line[0 : chunk.windowWidth-4]
				line = line + "..."
			}
		}
		cli.Println(line)
	}
}

func tableTypeDesc(typ int, flg int) string {
	desc := "undef"
	switch typ {
	case 0:
		desc = "Log Table"
	case 1:
		desc = "Fixed Table"
	case 3:
		desc = "Volatile Table"
	case 4:
		desc = "Lookup Table"
	case 5:
		desc = "KeyValue Table"
	case 6:
		desc = "Tag Table"
	}
	switch flg {
	case 1:
		desc += " (data)"
	case 2:
		desc += " (rollup)"
	case 4:
		desc += " (meta)"
	case 8:
		desc += " (stat)"
	}
	return desc
}
