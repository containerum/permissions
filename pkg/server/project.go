package server

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ProjectActions interface {
	CreateProject(ctx context.Context, label string) error
}

func (s *Server) CreateProject(ctx context.Context, label string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("create project")

	err := s.db.Transactional(func(tx database.DB) error {
		project := model.Project{
			Resource: model.Resource{
				OwnerUserID: userID,
				Label:       label,
			},
		}
		return tx.CreateProject(ctx, &project)
	})
	return err
}
