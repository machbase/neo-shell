package pretty

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func Module(rt *goja.Runtime, module *goja.Object) {
	// Export native functions
	exports := module.Get("exports").(*goja.Object)
	exports.Set("Table", Table)
	exports.Set("MakeRow", MakeRow)
	exports.Set("Bytes", Bytes)
	exports.Set("Ints", Ints)
	exports.Set("Durations", Durations)

	exports.Set("isTerminal", IsTerminal)
	exports.Set("getTerminalSize", GetTerminalSize)
	exports.Set("pauseTerminal", PauseTerminal)
}

func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

type TermSize struct {
	Width  int
	Height int
}

func (ts TermSize) String() string {
	return fmt.Sprintf("{Width: %d, Height: %d}", ts.Width, ts.Height)
}

func GetTerminalSize() (TermSize, error) {
	if x, y, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return TermSize{Width: x, Height: y}, nil
	} else {
		return TermSize{}, err
	}
}

// PauseTerminal waits for user to press a key. Returns false if user pressed 'q' or 'Q'.
// Otherwise returns true.
func PauseTerminal() bool {
	fmt.Fprintf(os.Stdout, ":")
	// switch stdin into 'raw' mode
	if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		var b []byte = make([]byte, 3)
		if _, err := os.Stdin.Read(b); err == nil {
			term.Restore(int(os.Stdin.Fd()), oldState)
			// remove prompt, erase the current line
			fmt.Fprintf(os.Stdout, "\x1b[2K")
			// cursor backward
			fmt.Fprintf(os.Stdout, "\x1b[1D")
			if b[0] == 'q' || b[0] == 'Q' {
				return false
			}
			return true
		}
		term.Restore(int(os.Stdin.Fd()), oldState)
	}
	return true
}

var (
	defaultLang language.Tag = language.English
)

func Bytes(v int64) string {
	p := message.NewPrinter(defaultLang)
	f := float64(v)
	u := ""
	switch {
	case v >= 1024*1024*1024*1024:
		f = f / (1024 * 1024 * 1024 * 1024)
		u = "TB"
	case v >= 1024*1024*1024:
		f = f / (1024 * 1024 * 1024)
		u = "GB"
	case v >= 1024*1024:
		f = f / (1024 * 1024)
		u = "MB"
	case v >= 1024:
		f = f / 1024
		u = "KB"
	default:
		return p.Sprintf("%dB", v)
	}
	return p.Sprintf("%.1f%s", f, u)
}

func Ints(v int64) string {
	p := message.NewPrinter(defaultLang)
	return p.Sprintf("%d", v)
}

func Durations(v time.Duration) string {
	p := message.NewPrinter(defaultLang)
	totalSeconds := int64(v.Seconds())
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	parts := []string{}
	if days > 0 {
		parts = append(parts, p.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, p.Sprintf("%dh", hours))
	}
	if minutes > 0 || hours > 0 || days > 0 {
		parts = append(parts, p.Sprintf("%dm", minutes))
	}
	parts = append(parts, p.Sprintf("%ds", seconds))
	return strings.Join(parts, " ")
}

type TableOption struct {
	BoxStyle   string `json:"boxStyle"`
	Timeformat string `json:"timeformat"`
	Tz         string `json:"tz"`
	Precision  int    `json:"precision"`
	Format     string `json:"format"`
	Header     bool   `json:"header"`
	Footer     bool   `json:"footer"`
	Pause      bool   `json:"pause"`
	Rownum     bool   `json:"rownum"`
}

type TableWriter struct {
	table.Writer
	format     string
	timeformat string
	tz         *time.Location
	precision  int
	headerRow  table.Row
	header     bool
	footer     bool
	pause      bool
	rownum     bool
	rowCount   int64

	nextPauseRow         int64
	pageHeight           int
	pageHeightSpaceLines int
	termSize             TermSize
}

func Table(opt TableOption) table.Writer {
	ret := &TableWriter{
		Writer:    table.NewWriter(),
		tz:        time.Local,
		precision: opt.Precision,
		format:    opt.Format,
		header:    opt.Header,
		footer:    opt.Footer,
		pause:     opt.Pause,
		rownum:    opt.Rownum,
	}
	ret.SetBoxStyle(opt.BoxStyle)
	ret.SetTimeformat(opt.Timeformat)
	ret.SetTz(opt.Tz)
	ret.SetAutoIndex(false)

	if ret.pause && IsTerminal() {
		if ts, err := GetTerminalSize(); err == nil {
			ret.termSize = ts
			// set next pause row threshold
			ret.pageHeight = ts.Height - 1 // leave one line for prompt
			if ret.header {
				ret.pageHeight -= ret.pageHeightSpaceLines // leave lines for header
			}
			ret.nextPauseRow = int64(ret.pageHeight)
		} else {
			ret.pause = false
		}
	}
	return ret
}

type BoxStyle struct {
	style          table.Style
	pageSpaceLines int
	option         func(*table.Style)
}

var boxStyles = map[string]BoxStyle{
	// Basic styles
	"LIGHT":   {table.StyleLight, 4, nil},
	"DOUBLE":  {table.StyleDouble, 4, nil},
	"BOLD":    {table.StyleBold, 4, nil},
	"ROUNDED": {table.StyleRounded, 4, nil},
	"ROUND":   {table.StyleRounded, 4, nil},
	"SIMPLE":  {table.StyleDefault, 4, nil},
	// Bright color styles
	"BRIGHT":         {table.StyleColoredBright, 1, nil},
	"BRIGHT_BLUE":    {table.StyleColoredBlackOnBlueWhite, 1, nil},
	"BRIGHT_CYAN":    {table.StyleColoredBlackOnCyanWhite, 1, nil},
	"BRIGHT_GREEN":   {table.StyleColoredBlackOnGreenWhite, 1, nil},
	"BRIGHT_MAGENTA": {table.StyleColoredBlackOnMagentaWhite, 1, nil},
	"BRIGHT_YELLOW":  {table.StyleColoredBlackOnYellowWhite, 1, nil},
	"BRIGHT_RED":     {table.StyleColoredBlackOnRedWhite, 1, nil},
	// Dark color styles
	"DARK":         {table.StyleColoredDark, 1, nil},
	"DARK_BLUE":    {table.StyleColoredBlueWhiteOnBlack, 1, nil},
	"DARK_CYAN":    {table.StyleColoredCyanWhiteOnBlack, 1, nil},
	"DARK_GREEN":   {table.StyleColoredGreenWhiteOnBlack, 1, nil},
	"DARK_MAGENTA": {table.StyleColoredMagentaWhiteOnBlack, 1, nil},
	"DARK_YELLOW":  {table.StyleColoredYellowWhiteOnBlack, 1, nil},
	"DARK_RED":     {table.StyleColoredRedWhiteOnBlack, 1, nil},
	// Compact style
	"COMPACT": {table.StyleLight, 2, func(s *table.Style) {
		s.Options.DrawBorder = false
		s.Options.SeparateColumns = false
	}},
}

func (tw *TableWriter) SetBoxStyle(style string) {
	styleUpper := strings.ToUpper(style)
	if o, ok := boxStyles[styleUpper]; ok {
		s := o.style
		if o.option != nil {
			o.option(&s)
		}
		tw.SetStyle(s)
		tw.pageHeightSpaceLines = o.pageSpaceLines
	} else {
		tw.SetStyle(table.StyleDefault)
		tw.pageHeightSpaceLines = 4
	}
}

func (tw *TableWriter) SetTimeformat(format string) {
	switch strings.ToUpper(format) {
	case "DEFAULT", "":
		tw.timeformat = "2006-01-02 15:04:05.999"
	case "DATETIME":
		tw.timeformat = time.DateTime
	case "DATE":
		tw.timeformat = time.DateOnly
	case "TIME":
		tw.timeformat = time.TimeOnly
	case "RFC3339":
		tw.timeformat = time.RFC3339Nano
	case "RFC1123":
		tw.timeformat = time.RFC1123
	case "ANSIC":
		tw.timeformat = time.ANSIC
	case "KITCHEN":
		tw.timeformat = time.Kitchen
	case "STAMP":
		tw.timeformat = time.Stamp
	case "STAMPMILLI":
		tw.timeformat = time.StampMilli
	case "STAMPMICRO":
		tw.timeformat = time.StampMicro
	case "STAMPNANO":
		tw.timeformat = time.StampNano
	default:
		tw.timeformat = format
	}
}

func (tw *TableWriter) SetTz(tz string) {
	switch strings.ToUpper(tz) {
	case "", "LOCAL":
		tw.tz = time.Local
	case "UTC":
		tw.tz = time.UTC
	default:
		if tz, err := time.LoadLocation(tz); err == nil {
			tw.tz = tz
		} else {
			panic(err)
		}
	}
}

func (tw *TableWriter) SetAutoIndex(autoIndex bool) {
	// always disable auto index feature favored over the rownum option
	tw.Writer.SetAutoIndex(false)
}

func (tw *TableWriter) AppendHeader(v table.Row, configs ...table.RowConfig) {
	tw.headerRow = v // store header row
	if !tw.header {
		return
	}
	if tw.rownum {
		v = append(table.Row{"ROWNUM"}, v...)
	}
	tw.Writer.AppendHeader(v, configs...)
}

func (tw *TableWriter) SetCaption(format string, a ...interface{}) {
	if !tw.footer {
		return
	}
	tw.Writer.SetCaption(format, a...)
}

func (tw *TableWriter) Append(v any, configs ...table.RowConfig) {
	switch v := v.(type) {
	case table.Row:
		tw.AppendRow(v, configs...)
	case []table.Row:
		tw.AppendRows(v, configs...)
	case []interface{}:
		row := tw.Row(v...)
		tw.AppendRow(row, configs...)
	default:
		return
	}
}

func (tw *TableWriter) AppendRow(row table.Row, configs ...table.RowConfig) {
	tw.rowCount++
	if tw.rownum {
		row = append(table.Row{tw.rowCount}, row...)
	}
	tw.Writer.AppendRow(row, configs...)
}

func (tw *TableWriter) RequirePageRender() bool {
	if tw.pause {
		return tw.nextPauseRow > 0 && tw.rowCount == tw.nextPauseRow
	} else {
		return tw.rowCount%1000 == 0
	}
}

// PauseAndWait pauses the table rendering and waits for user input.
// Returns false if user pressed 'q' or 'Q' to quit, otherwise returns true.
func (tw *TableWriter) PauseAndWait() bool {
	if !tw.pause {
		tw.ResetRows()    // clear the table rows
		tw.ResetHeaders() // do not render header again
		return true
	}
	// set next pause row threshold
	tw.nextPauseRow += int64(tw.pageHeight)
	// wait for user input
	continued := PauseTerminal()
	// clear the table rows
	tw.ResetRows()
	return continued
}

func (tw *TableWriter) AppendRows(rows []table.Row, configs ...table.RowConfig) {
	for _, row := range rows {
		tw.AppendRow(row, configs...)
	}
}

func (tw *TableWriter) Row(values ...interface{}) table.Row {
	for i, value := range values {
		switch val := value.(type) {
		case time.Time:
			values[i] = val.In(tw.tz).Format(tw.timeformat)
		case float32:
			if tw.precision >= 0 {
				factor := math.Pow(10, float64(tw.precision))
				values[i] = float32(math.Round(float64(val)*factor) / factor)
			}
		case float64:
			if tw.precision >= 0 {
				factor := math.Pow(10, float64(tw.precision))
				values[i] = math.Round(val*factor) / factor
			}
		default:
			values[i] = value
		}
	}
	tr := table.Row(values)
	return tr
}

func MakeRow(size int) []table.Row {
	rows := make([]table.Row, size)
	return rows
}

func (tw *TableWriter) Render() string {
	switch strings.ToUpper(tw.format) {
	case "CSV":
		return tw.Writer.RenderCSV()
	case "HTML":
		return tw.Writer.RenderHTML()
	case "MARKDOWN", "MD":
		return tw.Writer.RenderMarkdown()
	case "TSV":
		return tw.Writer.RenderTSV()
	default:
		return tw.Writer.Render()
	}
}
