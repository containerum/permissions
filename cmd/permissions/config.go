package main

import (
	"log"
	"os"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/sirupsen/logrus"

	_ "git.containerum.net/ch/permissions/pkg/migrations" // to run migrations
)

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
