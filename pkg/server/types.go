package server

import (
	"fmt"
	"io"
	"reflect"

	"git.containerum.net/ch/permissions/pkg/clients"
	"git.containerum.net/ch/permissions/pkg/database"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/sirupsen/logrus"
)

type Clients struct {
	Auth      clients.AuthClient
	User      clients.UserManagerClient
	Kube      clients.KubeAPIClient
	Resource  clients.ResourceServiceClient
	Billing   clients.BillingClient
	Volume    clients.VolumeManagerClient
	Solutions clients.SolutionsClient
}

func (c *Clients) Close() error {
	var errs []error
	rval := reflect.ValueOf(c)
	for i := 0; i < rval.NumField(); i++ {
		if closer, ok := rval.Field(i).Interface().(io.Closer); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("clients close errors: %v", errs)
	}

	return nil
}

type Server struct {
	db      database.DB
	log     *cherrylog.LogrusAdapter
	clients *Clients
}

func NewServer(db database.DB, clients *Clients) *Server {
	return &Server{
		db:      db,
		log:     cherrylog.NewLogrusAdapter(logrus.WithField("component", "entry")),
		clients: clients,
	}
}

func (s *Server) Close() error {
	return s.clients.Close()
}
