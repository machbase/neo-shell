package pretty

import (
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/jedib0t/go-pretty/v6/table"
)

func Module(rt *goja.Runtime, module *goja.Object) {
	// Export native functions
	exports := module.Get("exports").(*goja.Object)
	exports.Set("Table", Table)
	exports.Set("MakeRow", MakeRow)
}

type TableOption struct {
	Timeformat string `json:"timeformat"`
	TZ         string `json:"tz"`
	Precision  int    `json:"precision"`
}

type TableWriter struct {
	table.Writer
	timeformat string
	tz         *time.Location
	precision  int
}

func Table(opt TableOption) table.Writer {
	ret := &TableWriter{
		Writer:     table.NewWriter(),
		timeformat: opt.Timeformat,
		tz:         time.Local,
		precision:  opt.Precision,
	}

	switch strings.ToUpper(opt.Timeformat) {
	case "DEFAULT":
		ret.timeformat = "2006-01-02 15:04:05.999"
	case "DATETIME":
		ret.timeformat = time.DateTime
	case "DATE":
		ret.timeformat = time.DateOnly
	case "TIME":
		ret.timeformat = time.TimeOnly
	case "RFC3339":
		ret.timeformat = time.RFC3339Nano
	case "RFC1123":
		ret.timeformat = time.RFC1123
	case "ANSIC":
		ret.timeformat = time.ANSIC
	case "KITCHEN":
		ret.timeformat = time.Kitchen
	case "STAMP":
		ret.timeformat = time.Stamp
	case "STAMPMILLI":
		ret.timeformat = time.StampMilli
	case "STAMPMICRO":
		ret.timeformat = time.StampMicro
	case "STAMPNANO":
		ret.timeformat = time.StampNano
	}
	if opt.TZ != "" {
		ret.tz, _ = time.LoadLocation(opt.TZ)
	}
	return ret
}

func (tw *TableWriter) Row(values ...interface{}) table.Row {
	for i, value := range values {
		switch val := value.(type) {
		case time.Time:
			values[i] = val.In(tw.tz).Format(tw.timeformat)
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
