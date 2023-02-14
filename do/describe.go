package do

import (
	"strings"

	spi "github.com/machbase/neo-spi"
)

// Describe retrieves the result of 'desc table'.
//
// If includeHiddenColumns is true, the result includes hidden columns those name start with '_'
// such as "_RID" and "_ARRIVAL_TIME".
func Describe(db spi.Database, name string, includeHiddenColumns bool) (Description, error) {
	d := &TableDescription{}
	var tableType int
	var colCount int
	var colType int
	r := db.QueryRow("select name, type, flag, id, colcount from M$SYS_TABLES where name = ?", strings.ToUpper(name))
	if err := r.Scan(&d.Name, &tableType, &d.Flag, &d.Id, &colCount); err != nil {
		return nil, err
	}
	d.Type = spi.TableType(tableType)

	rows, err := db.Query(`
		select
			name, type, length, id
		from
			M$SYS_COLUMNS
		where
			table_id = ? 
		order by id`, d.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		col := &ColumnDescription{}
		err = rows.Scan(&col.Name, &colType, &col.Length, &col.Id)
		if err != nil {
			return nil, err
		}
		if !includeHiddenColumns && strings.HasPrefix(col.Name, "_") {
			continue
		}
		col.Type = spi.ColumnType(colType)
		d.Columns = append(d.Columns, col)
	}
	return d, nil
}

type Description interface {
	description()
}

func (td *TableDescription) description()  {}
func (cd *ColumnDescription) description() {}

// TableDescription is represents data that comes as a result of 'desc <table>'
type TableDescription struct {
	Name    string               `json:"name"`
	Type    spi.TableType        `json:"type"`
	Flag    int                  `json:"flag"`
	Id      int                  `json:"id"`
	Columns []*ColumnDescription `json:"columns"`
}

// TypeString returns string representation of table type.
func (td *TableDescription) TypeString() string {
	return TableTypeDescription(td.Type, td.Flag)
}

// TableTypeDescription converts the given TableType and flag into string representation.
func TableTypeDescription(typ spi.TableType, flag int) string {
	desc := "undef"
	switch typ {
	case spi.LogTableType:
		desc = "Log Table"
	case spi.FixedTableType:
		desc = "Fixed Table"
	case spi.VolatileTableType:
		desc = "Volatile Table"
	case spi.LookupTableType:
		desc = "Lookup Table"
	case spi.KeyValueTableType:
		desc = "KeyValue Table"
	case spi.TagTableType:
		desc = "Tag Table"
	}
	switch flag {
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

// columnDescription represents information of a column info.
type ColumnDescription struct {
	Id     uint64         `json:"id"`
	Name   string         `json:"name"`
	Type   spi.ColumnType `json:"type"`
	Length int            `json:"length"`
}

// TypeString returns string representation of column type.
func (cd *ColumnDescription) TypeString() string {
	return ColumnTypeDescription(cd.Type)
}

// ColumnTypeDescription converts ColumnType into string.
func ColumnTypeDescription(typ spi.ColumnType) string {
	switch typ {
	case spi.Int16ColumnType:
		return "int16"
	case spi.Uint16ColumnType:
		return "uint16"
	case spi.Int32ColumnType:
		return "int32"
	case spi.Uint32ColumnType:
		return "uint32"
	case spi.Int64ColumnType:
		return "int64"
	case spi.Uint64ColumnType:
		return "uint64"
	case spi.Float32ColumnType:
		return "float"
	case spi.Float64ColumnType:
		return "double"
	case spi.VarcharColumnType:
		return "varchar"
	case spi.TextColumnType:
		return "text"
	case spi.ClobColumnType:
		return "clob"
	case spi.BlobColumnType:
		return "blob"
	case spi.BinaryColumnType:
		return "binary"
	case spi.DatetimeColumnType:
		return "datetime"
	case spi.IpV4ColumnType:
		return "ipv4"
	case spi.IpV6ColumnType:
		return "ipv6"
	default:
		return "undef"
	}
}
