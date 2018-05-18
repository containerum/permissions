package main

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

var (
	ModeFlag = cli.StringFlag{
		Name:    "mode",
		EnvVars: []string{"MODE"},
		Value:   "debug",
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
	}

	UserAddrFlag = cli.StringFlag{
		Name:    "user_addr",
		EnvVars: []string{"USER_ADDR"},
	}

	KubeAPIAddrFlag = cli.StringFlag{
		Name:    "kube_api_addr",
		EnvVars: []string{"KUBE_API_ADDR"},
	}

	BillingAddrFlag = cli.StringFlag{
		Name:    "billing_addr",
		EnvVars: []string{"BILLING_ADDR"},
	}

	ResourceServiceAddrFlag = cli.StringFlag{
		Name:    "resource_service_addr",
		EnvVars: []string{"RESOURCE_SERVICE_ADDR"},
	}

	CORSFlag = cli.BoolFlag{
		Name: "cors",
	}
)
