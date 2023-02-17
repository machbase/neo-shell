package sshsvr

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gliderlabs/ssh"
	logging "github.com/machbase/neo-logging"
	"github.com/machbase/neo-shell/server/sshsvr/sshd"
	spi "github.com/machbase/neo-spi"
	"github.com/pkg/errors"
)

type Server interface {
	GetGrpcAddresses() []string
	ValidateSshPublicKey(keyType string, key string) bool
}

type Config struct {
	Listeners     []string
	IdleTimeout   time.Duration
	ServerKeyPath string
}

type MachShell struct {
	conf  *Config
	log   logging.Log
	sshds []sshd.Server

	db spi.Database

	versionString string
	gitSHA        string
	editionString string

	Server Server // injection point
}

func New(db spi.Database, conf *Config) *MachShell {
	return &MachShell{
		conf: conf,
		db:   db,
	}
}

func (svr *MachShell) Start() error {
	svr.log = logging.GetLog("neoshell")
	svr.sshds = make([]sshd.Server, 0)

	if svr.db == nil {
		return errors.New("no database instance")
	}
	if nfo, err := svr.db.GetServerInfo(); err != nil {
		return errors.Wrap(err, "no database info")
	} else {
		svr.editionString = nfo.Version.Engine
		svr.versionString = fmt.Sprintf("v%d.%d.%d",
			nfo.Version.Major, nfo.Version.Minor, nfo.Version.Patch)
		svr.gitSHA = nfo.Version.GitSHA
	}

	for _, listen := range svr.conf.Listeners {
		listenAddress := strings.TrimPrefix(listen, "tcp://")
		cfg := sshd.Config{
			ListenAddress:      listenAddress,
			ServerKey:          svr.conf.ServerKeyPath,
			IdleTimeout:        svr.conf.IdleTimeout,
			AutoListenAndServe: false,
		}
		s := sshd.New(&cfg)
		err := s.Start()
		if err != nil {
			return errors.Wrap(err, "machsell")
		}
		s.SetShellProvider(svr.shellProvider)
		s.SetMotdProvider(svr.motdProvider)
		s.SetPasswordHandler(svr.passwordProvider)
		s.SetPublicKeyHandler(svr.publicKeyProvider)
		go func() {
			err := s.ListenAndServe()
			if err != nil {
				svr.log.Warnf("machshell-listen %s", err.Error())
			}
		}()
		svr.log.Infof("SSHD Listen %s", listen)
	}
	return nil
}

func (svr *MachShell) Stop() {
	for _, s := range svr.sshds {
		s.Stop()
	}
}

func (svr *MachShell) shellProvider(user string) *sshd.Shell {
	grpcAddrs := svr.Server.GetGrpcAddresses()
	if len(grpcAddrs) == 0 {
		return nil
	}
	return &sshd.Shell{
		Cmd: os.Args[0],
		Args: []string{
			"shell",
			"--server", grpcAddrs[0],
			"--user", user,
		},
	}
}

func (svr *MachShell) motdProvider(user string) string {
	return fmt.Sprintf("Greetings, %s\r\nmachbase-neo %s (%s) %s\r\n",
		strings.ToUpper(user), svr.versionString, svr.gitSHA, svr.editionString)
}

func (svr *MachShell) passwordProvider(ctx ssh.Context, password string) bool {
	mdb, ok := svr.db.(spi.DatabaseAuth)
	if !ok {
		svr.log.Errorf("user auth - unknown database instance")
		return false
	}
	user := ctx.User()
	ok, err := mdb.UserAuth(user, password)
	if err != nil {
		svr.log.Errorf("user auth", err.Error())
		return false
	}
	return ok
}

func (svr *MachShell) publicKeyProvider(ctx ssh.Context, key ssh.PublicKey) bool {
	if svr.Server == nil {
		return false
	}
	return svr.Server.ValidateSshPublicKey(key.Type(), base64.StdEncoding.EncodeToString(key.Marshal()))
}
