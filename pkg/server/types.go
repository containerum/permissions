package server

import (
	"git.containerum.net/ch/kube-client/pkg/cherry"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/go-pg/pg"
	"github.com/sirupsen/logrus"
)

type Server struct {
	db  *pg.DB
	log *cherrylog.LogrusAdapter
}

func NewServer(db *pg.DB) *Server {
	return &Server{
		db:  db,
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "entry")),
	}
}

func (s *Server) handleTransactionError(err error) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *cherry.Err:
		return err
	default:
		return errors.ErrDatabase().Log(err, s.log)
	}
}
