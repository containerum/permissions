package server

import (
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
)

type Server struct {
	db  orm.DB
	log *cherrylog.LogrusAdapter
}

func NewServer(db orm.DB) *Server {
	return &Server{
		db:  db,
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "entry")),
	}
}
