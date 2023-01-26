package shell

import (
	"fmt"
	"io"
)

func (cli *client) Stdin() io.Reader {
	return cli.conf.Stdin
}
func (cli *client) Stdout() io.Writer {
	return cli.conf.Stdout
}
func (cli *client) Stderr() io.Writer {
	return cli.conf.Stderr
}

func (cli *client) Write(p []byte) (int, error) {
	return cli.conf.Stdout.Write(p)
}

func (cli *client) Print(args ...any) {
	fmt.Fprint(cli.conf.Stdout, args...)
}

func (cli *client) Printf(format string, args ...any) {
	str := fmt.Sprintf(format, args...)
	fmt.Fprint(cli.conf.Stdout, str)
}

func (cli *client) Println(args ...any) {
	fmt.Fprintln(cli.conf.Stdout, args...)
}

func (cli *client) Printfln(format string, args ...any) {
	fmt.Fprintf(cli.conf.Stdout, format+"\r\n", args...)
}
