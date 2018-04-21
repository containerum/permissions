package main

import (
	"errors"

	"git.containerum.net/ch/permissions/pkg/dao"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/universal-translator"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"

	_ "git.containerum.net/ch/permissions/pkg/migrations" // to run migrations
)

type operationMode int

const (
	modeDebug operationMode = iota
	modeRelease
)

var opMode operationMode

func setupLogger(ctx *cli.Context) error {
	mode := ctx.String(ModeFlag.Name)
	switch mode {
	case "debug":
		opMode = modeDebug
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
	case "release", "":
		opMode = modeRelease
		gin.SetMode(gin.ReleaseMode)
		logrus.SetFormatter(&logrus.JSONFormatter{})

		level := logrus.Level(ctx.Int(LogLevelFlag.Name))
		if level > logrus.DebugLevel || level < logrus.PanicLevel {
			return errors.New("invalid log level")
		}
		logrus.SetLevel(level)
	default:
		return errors.New("invalid operation mode (must be 'debug' or 'release')")
	}
	return nil
}

func setupDB(ctx *cli.Context) (*dao.DAO, error) {
	return dao.SetupDAO(ctx.String(DBAddrFlag.Name))
}

func getListenAddr(ctx *cli.Context) (string, error) {
	return ctx.String(ListenAddrFlag.Name), nil
}

func setupTranslator() *ut.UniversalTranslator {
	return ut.New(en.New(), en.New(), en_US.New())
}
