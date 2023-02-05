package shell

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/mgmt"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "key",
		PcFunc: pcKey,
		Action: doKey,
		Desc:   "Manage client keys",
		Usage:  helpKey,
	})
}

const helpKey = `  key command [options] [args...]
    commands:
      list        list registered keys
      del <id>    delete key
      gen <id>    generate new key with given id
`

type KeyCmd struct {
	List struct{} `cmd:"" name:"list"`
	Del  struct {
		KeyId string `arg:"" name:"id"`
	} `cmd:"" name:"del"`
	Gen struct {
		KeyId string `arg:"" name:"id"`
	} `cmd:"" name:"gen"`
	Help bool `kong:"-"`
}

func pcKey(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("key")
}

func doKey(cli Client, cmdLine string) {
	cmd := &KeyCmd{}
	parser, err := Kong(cmd, func() error { cli.Println(helpKey); cmd.Help = true; return nil })
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	ctx, err := parser.Parse(splitFields(cmdLine, false))
	if cmd.Help {
		return
	}
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	switch ctx.Command() {
	case "list":
		doKeyList(cli)
	case "gen <id>":
		doKeyGen(cli, cmd.Gen.KeyId)
	case "del <id>":
		doKeyDel(cli, cmd.Del.KeyId)
	default:
		cli.Println("ERR", fmt.Sprintf("unhandled command %s", ctx.Command()))
		return
	}
}

func doKeyList(cli Client) {
	mgmtCli, err := cli.(*client).NewManagementClient()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rsp, err := mgmtCli.ListKey(ctx, &mgmt.ListKeyRequest{})
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	if !rsp.Success {
		cli.Println("ERR", rsp.Reason)
		return
	}

	box := cli.NewBox([]string{"#", "ID", "VALID FROM", "EXPIRE"})
	for i, k := range rsp.Keys {
		nb := time.Unix(k.NotBefore, 0).UTC()
		na := time.Unix(k.NotAfter, 0).UTC()
		box.AppendRow(i+1, k.Id, nb.String(), na.String())
	}
	box.Render()
}

func doKeyDel(cli Client, id string) {
	mgmtCli, err := cli.(*client).NewManagementClient()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	rsp, err := mgmtCli.DelKey(ctx, &mgmt.DelKeyRequest{
		Id: id,
	})
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	if !rsp.Success {
		cli.Println("ERR", rsp.Reason)
		return
	}
	cli.Println("deleted")
}

func doKeyGen(cli Client, name string) {
	name = strings.ToLower(name)
	pass, _ := regexp.MatchString("[a-z][a-z0-9_.@-]+", name)
	if !pass {
		cli.Println("id contains invalid letter, use only alphnum and _.@-")
		return
	}

	mgmtCli, err := cli.(*client).NewManagementClient()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	rsp, err := mgmtCli.GenKey(ctx, &mgmt.GenKeyRequest{
		Id:        name,
		Type:      "ec",
		NotBefore: time.Now().Unix(),
		NotAfter:  time.Now().Add(10 * time.Hour * 24 * 365).Unix(),
	})
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	if !rsp.Success {
		cli.Println("ERR", rsp.Reason)
		return
	}

	cli.Println(rsp.Certificate)
	cli.Println(rsp.Key)
	cli.Println("-----BEGIN TOKEN-----")
	cli.Println(rsp.Token)
	cli.Println("-----END TOKEN-----")
	cli.Println("\nCaution:\n  This is the last chance to copy and store PRIVATE KEY and TOKEN.")
	cli.Println("  It can not be redo.")
}
