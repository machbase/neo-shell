package shell

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/gdamore/tcell/v2"
	"github.com/machbase/neo-grpc/spi"
	"github.com/machbase/neo-shell/util"
	"github.com/rivo/tview"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "walk",
		PcFunc: pcWalk,
		Action: doWalk,
		Desc:   "Execute query then walk-through the results",
		Usage:  helpWalk,
	})
}

const helpWalk = ` walk [options] <sql query>
  options:
     --[no-]rownum        show rownum
     --precision <int>    precision for float values
     --timeformat,-t      time format [ns|ms|s|<timeformat>] (default:'ns')
       ns, us, ms, s
         represents unix epoch time in nano-, micro-, milli- and seconds for each
       timeformat
         consult "help timeformat"
     --tz                  timezone for handling datetime`

type WalkCmd struct {
	TimeLocation *time.Location `name:"tz" default:"UTC"`
	Timeformat   string         `name:"timeformat" short:"t" default:"default"`
	Rownum       bool           `name:"rownum" negatable:"" default:"true"`
	Precision    int            `name:"precision" short:"p" default:"-1"`
	Help         bool           `kong:"-"`
	Query        []string       `arg:"" name:"query" passthrough:""`
}

func pcWalk(c Client) readline.PrefixCompleterInterface {
	cli := c.(*client)
	return readline.PcItem("walk", readline.PcItemDynamic(cli.SqlHistory))
}

func doWalk(cc Client, cmdLine string) {
	cmd := &WalkCmd{}
	parser, err := Kong(cmd, func() error { cc.Println(helpWalk); cmd.Help = true; return nil })
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}
	_, err = parser.Parse(splitFields(cmdLine, false))
	if cmd.Help {
		return
	}
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}

	sqlText := util.StripQuote(strings.Join(cmd.Query, " "))
	db := cc.Database()

	walker, err := NewWalker(sqlText, db, spi.GetTimeformat(cmd.Timeformat), cmd.TimeLocation, cmd.Precision)
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}
	defer walker.Close()

	app := tview.NewApplication()
	table := tview.NewTable()
	table.SetBorder(true).SetTitle(" ESC to quit, [yellow::bl]R[-::-]eload ").SetTitleAlign(tview.AlignLeft)
	table.SetFixed(1, 1)
	table.SetContent(walker)
	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyESC {
			app.Stop()
		}
	})
	table.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Rune() == 'r' || evt.Rune() == 'R' {
			walker.Reload()
			table.ScrollToBeginning()
			return nil
		}
		return evt
	})
	if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
		cc.Println("ERR", err.Error())
		return
	}
}

type Walker struct {
	tview.TableContentReadOnly
	sqlText    string
	db         spi.Database
	mutex      sync.Mutex
	rows       spi.Rows
	cols       spi.Columns
	values     [][]string
	eof        bool
	fetchSize  int
	tz         *time.Location
	timeformat string
	precision  int
}

func NewWalker(sqlText string, database spi.Database, timeformat string, tz *time.Location, precision int) (*Walker, error) {
	w := &Walker{
		sqlText:    sqlText,
		db:         database,
		fetchSize:  400,
		timeformat: timeformat,
		tz:         tz,
		precision:  precision,
	}
	return w, w.Reload()
}

func (w *Walker) Close() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.rows != nil {
		w.rows.Close()
		w.rows = nil
	}
}

func (w *Walker) Reload() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.rows != nil {
		w.rows.Close()
		w.rows = nil
	}

	rows, err := w.db.Query(w.sqlText)
	if err != nil {
		return err
	}

	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return err
	}

	values := make([][]string, 1)
	values[0] = make([]string, len(cols)+1)
	values[0][0] = "#"
	for i := range cols {
		if cols[i].Type == "datetime" {
			values[0][i+1] = fmt.Sprintf("%s(%s)", cols[i].Name, w.tz.String())
		} else {
			values[0][i+1] = cols[i].Name
		}
	}

	w.rows = rows
	w.cols = cols
	w.values = values
	w.eof = false
	return nil
}

func (w *Walker) GetCell(row, col int) *tview.TableCell {
	w.mutex.Lock()
	defer w.mutex.Unlock()

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

	buffer := w.cols.MakeBuffer()

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

		values := makeValues(buffer, w.tz, w.timeformat, w.precision)
		w.values = append(w.values, append([]string{strconv.Itoa(nrows + count)}, values...))

		count++
		if count >= w.fetchSize {
			return
		}
	}
}

func (w *Walker) GetRowCount() int {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.eof {
		return len(w.values)
	}
	return math.MaxInt32
}

func (w *Walker) GetColumnCount() int {
	return len(w.cols) + 1
}

func makeValues(rec []any, tz *time.Location, timeformat string, precision int) []string {
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
			cols[i] = v.In(tz).Format(timeformat)
		case *float64:
			if precision < 0 {
				cols[i] = fmt.Sprintf("%f", *v)
			} else {
				cols[i] = fmt.Sprintf("%.*f", precision, *v)
			}
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
