package main

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

var (
	ModeFlag = cli.StringFlag{
		Name:    "mode",
		EnvVars: []string{"MODE"},
		Value:   "release",
	}

	LogLevelFlag = cli.IntFlag{
		Name:    "log_level",
		EnvVars: []string{"LOG_LEVEL"},
		Value:   int(logrus.InfoLevel),
	}

	DBAddrFlag = cli.StringFlag{
		Name:    "db_url",
		EnvVars: []string{"DB_URL"},
		Value:   "postgres://postgres@localhost:5432/postgres?sslmode=disabled",
	}

	ListenAddrFlag = cli.StringFlag{
		Name:    "listen_addr",
		EnvVars: []string{"LISTEN_ADDR"},
		Value:   ":8080",
	}
)
