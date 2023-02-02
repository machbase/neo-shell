package shell

import "github.com/chzyer/readline"

func init() {
	RegisterCmd(&Cmd{
		Name:   "shutdown",
		PcFunc: pcShutdown,
		Action: doShutdown,
		Desc:   "Shutdown server",
		Usage:  helpShutdown,
	})
}

const helpShutdown string = `  shutdown`

type ShutdownCmd struct {
	Interactive bool `kong:"-"`
	Help        bool `kong:"-"`
}

func pcShutdown(cc Client) readline.PrefixCompleterInterface {
	return readline.PcItem("shutdown")
}

func doShutdown(cc Client, cmdLine string) {
	cli := cc.(*client)
	err := cli.ShutdownServer()
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}

	cc.Println("server shutting down...")
}
