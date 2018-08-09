package postgres

import (
	"io"
	"strings"
	"time"

	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"

	_ "git.containerum.net/ch/permissions/pkg/database/postgres/migrations" // to run migrations
)

type PgDB struct {
	db  orm.DB
	log *cherrylog.LogrusAdapter
}

func Connect(dbURL string) (*PgDB, error) {
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

	migrations.SetTableName("migrations_perm")
	oldVer, newVer, err := migrations.Run(db, "up")
	logrus.WithError(err).WithFields(logrus.Fields{
		"addr":    options.Addr,
		"old_ver": oldVer,
		"new_ver": newVer,
	}).Info("migrate up")

	return &PgDB{
		db:  db,
		log: cherrylog.NewLogrusAdapter(entry),
	}, err
}

type transactional interface {
	RunInTransaction(fn func(*pg.Tx) error) error
}

func (pgdb *PgDB) handleError(err error) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *cherry.Err:
		return err
	default:
		return errors.ErrInternal().Log(err, pgdb.log)
	}
}

func (pgdb *PgDB) Transactional(fn func(tx database.DB) error) error {
	entry := cherrylog.NewLogrusAdapter(pgdb.log.WithField("transaction_id", time.Now().UTC().Unix()))
	dtx := &PgDB{log: entry}
	err := pgdb.db.(transactional).RunInTransaction(func(tx *pg.Tx) error {
		dtx.db = tx
		return fn(dtx)
	})

	return dtx.handleError(err)
}

func (pgdb *PgDB) Close() error {
	if cl, ok := pgdb.db.(io.Closer); ok {
		return cl.Close()
	}
	return nil
}
