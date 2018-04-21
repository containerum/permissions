package server

import (
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/permissions/pkg/dao"
	"github.com/sirupsen/logrus"
)

type Server struct {
	db  *dao.DAO
	log *cherrylog.LogrusAdapter
}

func NewServer(db *dao.DAO) *Server {
	return &Server{
		db:  db,
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "entry")),
	}
}
