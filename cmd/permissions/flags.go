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
		Value:   ":4242",
	}

	AuthAddrFlag = cli.StringFlag{
		Name:    "auth_addr",
		EnvVars: []string{"AUTH_ADDR"},
		Value:   "localhost:8888",
	}

	UserAddrFlag = cli.StringFlag{
		Name:    "user_addr",
		EnvVars: []string{"USER_ADDR"},
		Value:   "localhost:8111",
	}

	KubeAPIAddrFlag = cli.StringFlag{
		Name:    "kube_api_addr",
		EnvVars: []string{"KUBE_API_ADDR"},
		Value:   "localhost:1212",
	}

	BillingAddrFlag = cli.StringFlag{
		Name:    "billing_addr",
		EnvVars: []string{"BILLING_ADDR"},
		Value:   "localhost:5000",
	}

	CORSFlag = cli.BoolFlag{
		Name: "cors",
	}
)
