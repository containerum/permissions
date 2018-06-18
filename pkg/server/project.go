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
	AddGroup(ctx context.Context, project, groupID string) error
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

func (s *Server) AddGroup(ctx context.Context, project, groupID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"group_id": groupID,
		"project":  project,
	}).Info("add group")

	group, err := s.clients.User.Group(ctx, groupID)
	if err != nil {
		return err
	}

	var accessList []database.AccessListElement
	for _, v := range group.Members {
		accessList = append(accessList, database.AccessListElement{
			AccessLevel: UserGroupAccessToDBAccess(v.Access),
			ToUserID:    v.Username,
		})
	}

	err = s.db.Transactional(func(tx database.DB) error {
		project, getErr := tx.ProjectByID(ctx, project)
		if getErr != nil {
			return getErr
		}

		return tx.SetNamespacesAccesses(ctx, project.Namespaces, accessList)
	})

	return err
}
