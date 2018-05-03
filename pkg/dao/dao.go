package dao

import (
	"strings"
	"time"

	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"

	_ "git.containerum.net/ch/permissions/pkg/migrations" // to run migrations
)

type DAO struct {
	db  orm.DB
	log *cherrylog.LogrusAdapter
}

func SetupDAO(dbURL string) (*DAO, error) {
	options, err := pg.ParseURL(dbURL)
	if err != nil {
		return nil, err
	}

	db := pg.Connect(options)

	entry := logrus.WithField("component", "db")

	db.OnQueryProcessed(func(event *pg.QueryProcessedEvent) {
		query, err := event.FormattedQuery()
		if err != nil {
			entry = entry.WithError(err)
		}
		query = strings.Join(strings.Fields(query), " ") // drop "\n", "\t" and exceeded spaces
		entry.WithField("query", query).Debugf("DB Query")
	})

	entry.WithField("addr", options.Addr).Info("run migrations")

	oldVer, newVer, err := migrations.Run(db, "up")
	logrus.WithError(err).WithFields(logrus.Fields{
		"addr":    options.Addr,
		"old_ver": oldVer,
		"new_ver": newVer,
	}).Info("migrate up")

	return &DAO{
		db:  db,
		log: cherrylog.NewLogrusAdapter(entry),
	}, err
}

type transactional interface {
	RunInTransaction(fn func(*pg.Tx) error) error
}

func (dao *DAO) handleError(err error) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *cherry.Err:
		return err
	default:
		return errors.ErrInternal().Log(err, dao.log)
	}
}

func (dao *DAO) Transactional(fn func(tx *DAO) error) error {
	entry := cherrylog.NewLogrusAdapter(dao.log.WithField("transaction_id", time.Now().UTC().Unix()))
	dtx := &DAO{log: entry}
	err := dao.db.(transactional).RunInTransaction(func(tx *pg.Tx) error {
		dtx.db = tx
		return fn(dtx)
	})

	return dtx.handleError(err)
}