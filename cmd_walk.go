package shell

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/gdamore/tcell/v2"
	"github.com/machbase/neo-grpc/machrpc"
	"github.com/rivo/tview"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "walk",
		Aliases: []string{"\\w"},
		PcFunc:  pcWalk,
		Action:  doWalk,
		Desc:    "execute query then walk-through the results",
		Usage:   "  walk <sql query>",
	})
}

func pcWalk(c Client) readline.PrefixCompleterInterface {
	return readline.PcItem("walk")
}

func doWalk(cc Client, sqlText string, interactive bool) {
	cli := cc.(*client)
	sqlText = strings.TrimSpace(sqlText)
	if len(sqlText) == 0 {
		cli.Println("Usage: walk <sql query>")
		return
	}

	walker, err := NewWalker(sqlText, cli.db, cli.conf.TimeLocation)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	defer walker.Close()

	app := tview.NewApplication()
	table := tview.NewTable()
	table.SetBorder(true).SetTitle(" ESC to quit ").SetTitleAlign(tview.AlignLeft)
	table.SetFixed(1, 1)
	table.SetContent(walker)
	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyESC {
			app.Stop()
		}
	})
	if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
		cli.Println("ERR", err.Error())
		return
	}
}

type Walker struct {
	tview.TableContentReadOnly
	db        *machrpc.Client
	rows      *machrpc.Rows
	cols      []*machrpc.Column
	values    [][]string
	eof       bool
	fetchSize int
	tz        *time.Location
}

func NewWalker(sqlText string, client *machrpc.Client, tz *time.Location) (*Walker, error) {
	rows, err := client.Query(sqlText)
	if err != nil {
		return nil, err
	}

	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, err
	}

	values := make([][]string, 1)
	values[0] = make([]string, len(cols)+1)
	values[0][0] = "#"
	for i := range cols {
		if cols[i].Type == "datetime" {
			values[0][i+1] = fmt.Sprintf("%s(%s)", cols[i].Name, tz.String())
		} else {
			values[0][i+1] = cols[i].Name
		}
	}

	return &Walker{
		db:        client,
		rows:      rows,
		cols:      cols,
		values:    values,
		fetchSize: 400,
		tz:        tz,
	}, nil
}

func (w *Walker) Close() {
	if w.rows != nil {
		w.rows.Close()
		w.rows = nil
	}
}

func (w *Walker) GetCell(row, col int) *tview.TableCell {
	if row == 0 {
		return tview.NewTableCell(w.values[row][col]).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter)
	}

	if row >= len(w.values) {
		w.fetchMore()
	}

	if row < len(w.values) {
		color := tcell.ColorWhite
		if col == 0 {
			color = tcell.ColorYellow
		}
		return tview.NewTableCell(w.values[row][col]).SetTextColor(color)
	} else {
		return nil
	}
}

func (w *Walker) fetchMore() {
	if w.eof {
		return
	}

	buffer := makeBuffer(w.cols)

	count := 0
	nrows := len(w.values)
	for {
		if !w.rows.Next() {
			w.eof = true
			return
		}

		err := w.rows.Scan(buffer...)
		if err != nil {
			w.eof = true
			return
		}

		values := makeValues(buffer, w.tz)
		w.values = append(w.values, append([]string{strconv.Itoa(nrows + count)}, values...))

		count++
		if count >= w.fetchSize {
			return
		}
	}
}

func (w *Walker) GetRowCount() int {
	if w.eof {
		return len(w.values)
	}
	return math.MaxInt64
}

func (w *Walker) GetColumnCount() int {
	return len(w.cols) + 1
}
