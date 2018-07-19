package server

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/model"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ProjectActions interface {
	CreateProject(ctx context.Context, label string) error
	AddGroup(ctx context.Context, project, groupID string) error
	GetProjectGroups(ctx context.Context, projectID string) ([]kubeClientModel.UserGroup, error)
	SetGroupMemberAccess(ctx context.Context, projectID, groupID string, req model.SetGroupMemberAccessRequest) error
	DeleteGroupFromProject(ctx context.Context, projectID, groupID string) error
	AddMemberToProject(ctx context.Context, projectID string, req model.AddMemberToProjectRequest) error
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
			AccessLevel: v.Access,
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

func (s *Server) GetProjectGroups(ctx context.Context, projectID string) ([]kubeClientModel.UserGroup, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"project_id": projectID,
		"user_id":    userID,
	}).Infof("get project groups")

	project, err := s.db.ProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if len(project.Namespaces) == 0 {
		return make([]kubeClientModel.UserGroup, 0), nil
	}

	nsWithPermissions := make([]model.NamespaceWithPermissions, len(project.Namespaces))
	for i := range project.Namespaces {
		nsWithPermissions[i].Namespace = project.Namespaces[i]
		err = s.db.NamespacePermissions(ctx, &nsWithPermissions[i])
		if err != nil {
			return nil, err
		}
	}

	var groupIDs []string
	for _, ns := range nsWithPermissions {
		for _, v := range ns.Permissions {
			if v.GroupID != nil {
				groupIDs = append(groupIDs, *v.GroupID)
			}
		}
	}

	groups, err := s.clients.User.GroupFullIDList(ctx, groupIDs...)
	if err != nil {
		return nil, err
	}

	return groups.Groups, nil
}

func (s *Server) SetGroupMemberAccess(ctx context.Context, projectID, groupID string, req model.SetGroupMemberAccessRequest) error {
	s.log.WithFields(logrus.Fields{
		"project":  projectID,
		"group":    groupID,
		"username": req.Username,
		"access":   req.AccessLevel,
	}).Infof("set group member access")

	err := s.db.Transactional(func(tx database.DB) error {
		project, getErr := tx.ProjectByID(ctx, projectID)
		if getErr != nil {
			return getErr
		}

		user, getErr := s.clients.User.UserInfoByLogin(ctx, req.Username)
		if getErr != nil {
			return getErr
		}

		accesses := []database.AccessListElement{
			{ToUserID: user.ID, AccessLevel: req.AccessLevel},
		}
		if setErr := tx.SetNamespacesAccesses(ctx, project.Namespaces, accesses); setErr != nil {
			return setErr
		}

		return updateUserAccesses(ctx, s.clients.Auth, s.db, user.ID)
	})

	return err
}

func (s *Server) DeleteGroupFromProject(ctx context.Context, projectID, groupID string) error {
	s.log.WithFields(logrus.Fields{
		"project": projectID,
		"group":   groupID,
	}).Infof("delete group from project")

	err := s.db.Transactional(func(tx database.DB) error {
		delPerms, delErr := tx.DeleteGroupFromProject(ctx, projectID, groupID)
		if delErr != nil {
			return delErr
		}
		users := make(map[string]bool)
		for _, v := range delPerms {
			users[v.UserID] = true
		}

		for user := range users {
			if updErr := updateUserAccesses(ctx, s.clients.Auth, s.db, user); updErr != nil {
				s.log.WithError(updErr).Warnf("update access failed for user %s", user)
			}
		}

		return nil
	})

	return err
}

func (s *Server) AddMemberToProject(ctx context.Context, projectID string, req model.AddMemberToProjectRequest) error {
	s.log.WithFields(logrus.Fields{
		"project_id": projectID,
		"username":   req.Username,
		"access":     req.AccessLevel,
	}).Infof("add member to project")

	user, err := s.clients.User.UserInfoByLogin(ctx, req.Username)
	if err != nil {
		return err
	}

	err = s.db.Transactional(func(tx database.DB) error {
		project, getErr := tx.ProjectByID(ctx, projectID)
		if getErr != nil {
			return getErr
		}

		access := []database.AccessListElement{
			{ToUserID: user.ID, AccessLevel: req.AccessLevel},
		}
		if setErr := tx.SetNamespacesAccesses(ctx, project.Namespaces, access); setErr != nil {
			return setErr
		}

		return updateUserAccesses(ctx, s.clients.Auth, s.db, user.ID)
	})

	return err
}
