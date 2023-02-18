package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/mgmt"
	"github.com/machbase/neo-shell/client"
	"github.com/machbase/neo-shell/util"
)

func init() {
	client.RegisterCmd(&client.Cmd{
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

func pcKey() readline.PrefixCompleterInterface {
	return readline.PcItem("key")
}

func doKey(ctx *client.ActionContext) {
	cmd := &KeyCmd{}
	parser, err := client.Kong(cmd, func() error { ctx.Println(helpKey); cmd.Help = true; return nil })
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	parseCtx, err := parser.Parse(util.SplitFields(ctx.Line, false))
	if cmd.Help {
		return
	}
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}

	switch parseCtx.Command() {
	case "list":
		doKeyList(ctx)
	case "gen <id>":
		doKeyGen(ctx, cmd.Gen.KeyId)
	case "del <id>":
		doKeyDel(ctx, cmd.Del.KeyId)
	default:
		ctx.Println("ERR", fmt.Sprintf("unhandled command %s", parseCtx.Command()))
		return
	}
}

func doKeyList(ctx *client.ActionContext) {
	mgmtCli, err := ctx.NewManagementClient()
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	rsp, err := mgmtCli.ListKey(ctx, &mgmt.ListKeyRequest{})
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}

	if !rsp.Success {
		ctx.Println("ERR", rsp.Reason)
		return
	}

	box := ctx.NewBox([]string{"ROWNUM", "ID", "VALID FROM", "EXPIRE"})
	for i, k := range rsp.Keys {
		nb := time.Unix(k.NotBefore, 0).UTC()
		na := time.Unix(k.NotAfter, 0).UTC()
		box.AppendRow(i+1, k.Id, nb.String(), na.String())
	}
	box.Render()
}

func doKeyDel(ctx *client.ActionContext, id string) {
	mgmtCli, err := ctx.NewManagementClient()
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	rsp, err := mgmtCli.DelKey(ctx, &mgmt.DelKeyRequest{
		Id: id,
	})
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	if !rsp.Success {
		ctx.Println("ERR", rsp.Reason)
		return
	}
	ctx.Println("deleted")
}

func doKeyGen(ctx *client.ActionContext, name string) {
	name = strings.ToLower(name)
	pass, _ := regexp.MatchString("[a-z][a-z0-9_.@-]+", name)
	if !pass {
		ctx.Println("id contains invalid letter, use only alphnum and _.@-")
		return
	}

	mgmtCli, err := ctx.NewManagementClient()
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	rsp, err := mgmtCli.GenKey(ctx, &mgmt.GenKeyRequest{
		Id:        name,
		Type:      "ec",
		NotBefore: time.Now().Unix(),
		NotAfter:  time.Now().Add(10 * time.Hour * 24 * 365).Unix(),
	})
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	if !rsp.Success {
		ctx.Println("ERR", rsp.Reason)
		return
	}

	ctx.Println(rsp.Certificate)
	ctx.Println(rsp.Key)
	ctx.Println("-----BEGIN TOKEN-----")
	ctx.Println(rsp.Token)
	ctx.Println("-----END TOKEN-----")
	ctx.Println("\nCaution:\n  This is the last chance to copy and store PRIVATE KEY and TOKEN.")
	ctx.Println("  It can not be redo.")
}
