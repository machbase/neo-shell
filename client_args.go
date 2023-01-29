package shell

import (
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/alecthomas/kong"
)

type TimezoneParser struct {
}

// implements kong.TypeMapper
func (tp *TimezoneParser) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	token, err := ctx.Scan.PopValue("tz")
	if err != nil {
		return err
	}
	tz := token.String()
	if strings.ToLower(tz) == "local" {
		target.Set(reflect.ValueOf(time.Local))
		return nil
	} else if tz == "UTC" {
		target.Set(reflect.ValueOf(time.UTC))
		return nil
	}
	if tz, err := time.LoadLocation(tz); err != nil {
		return err
	} else {
		target.Set(reflect.ValueOf(tz))
	}
	return nil
}

func KongOptions(helpFunc func() error) []kong.Option {
	return []kong.Option{
		kong.HelpOptions{Compact: true},
		kong.Exit(func(int) {}),
		kong.TypeMapper(reflect.TypeOf((*time.Location)(nil)), &TimezoneParser{}),
		kong.Help(func(options kong.HelpOptions, ctx *kong.Context) error { return helpFunc() }),
	}
}

func Kong(grammar any, helpFunc func() error) (*kong.Kong, error) {
	return kong.New(grammar, KongOptions(helpFunc)...)
}

// /////////////////
// utilites
func splitFields(line string, stripQuote bool) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}
	fields := strings.FieldsFunc(line, f)

	if stripQuote {
		for i, f := range fields {
			c := []rune(f)[0]
			if unicode.In(c, unicode.Quotation_Mark) {
				fields[i] = strings.Trim(f, string(c))
			}
		}
	}
	return fields
}

func stripQuote(str string) string {
	if len(str) == 0 {
		return str
	}
	c := []rune(str)[0]
	if unicode.In(c, unicode.Quotation_Mark) {
		return strings.Trim(str, string(c))
	}
	return str
}
