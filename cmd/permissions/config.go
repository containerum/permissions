package main

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/universal-translator"
	"github.com/sirupsen/logrus"

	_ "git.containerum.net/ch/permissions/pkg/migrations" // to run migrations
)

type operationMode int

const (
	modeDebug operationMode = iota
	modeRelease
)

var opMode operationMode

func setupLogger() error {
	mode := os.Getenv("MODE")
	switch mode {
	case "debug":
		opMode = modeDebug
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
	case "release", "":
		opMode = modeRelease
		gin.SetMode(gin.ReleaseMode)
		logrus.SetFormatter(&logrus.JSONFormatter{})

		logLevelString := os.Getenv("LOG_LEVEL")
		var level logrus.Level
		if logLevelString == "" {
			level = logrus.InfoLevel
		} else {
			levelI, err := strconv.Atoi(logLevelString)
			if err != nil {
				return err
			}
			level = logrus.Level(levelI)
			if level > logrus.DebugLevel || level < logrus.PanicLevel {
				return errors.New("invalid log level")
			}
		}
		logrus.SetLevel(level)
	default:
		return errors.New("invalid operation mode (must be 'debug' or 'release')")
	}
	return nil
}

func setupDB() (*pg.DB, error) {
	options, err := pg.ParseURL(os.Getenv("DB_URL"))
	if err != nil {
		return nil, err
	}

	pgLog := log.Logger{}
	pgLog.SetOutput(logrus.WithField("component", "db").WriterLevel(logrus.DebugLevel))
	pg.SetLogger(&pgLog)

	db := pg.Connect(options)

	db.OnQueryProcessed(func(event *pg.QueryProcessedEvent) {
		entry := logrus.WithField("component", "db")
		query, err := event.FormattedQuery()
		if err != nil {
			entry = entry.WithError(err)
		}
		entry.WithField("query", query).Debugf("Args: %v", event.Params)
	})

	logrus.WithField("addr", options.Addr).Info("run migrations")

	oldVer, newVer, err := migrations.Run(db, "up")
	logrus.WithError(err).WithFields(logrus.Fields{
		"addr":    options.Addr,
		"old_ver": oldVer,
		"new_ver": newVer,
	}).Info("migrate up")
	return db, err
}

func getListenAddr() (la string, err error) {
	if la = os.Getenv("LISTEN_ADDR"); la == "" {
		return "", errors.New("environment LISTEN_ADDR is not specified")
	}
	return la, nil
}

func setupTranslator() *ut.UniversalTranslator {
	return ut.New(en.New(), en.New(), en_US.New())
}
