package server

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type AccessActions interface {
	GetUserAccesses(ctx context.Context) ([]httputil.ProjectAccess, error)
	SetUserAccesses(ctx context.Context, accessLevel kubeClientModel.UserGroupAccess) error
	GetNamespaceAccesses(ctx context.Context, id string) (kubeClientModel.Namespace, error)
	GetNamespaceAccess(ctx context.Context, id string) (httputil.NamespaceAccess, error)
	SetNamespaceAccess(ctx context.Context, id, targetUser string, accessLevel kubeClientModel.UserGroupAccess) error
	DeleteNamespaceAccess(ctx context.Context, id string, targetUser string) error
}

func extractAccessesFromDB(ctx context.Context, db database.DB, userID string) ([]httputil.ProjectAccess, error) {
	userPermissions, err := db.UserAccesses(ctx, userID)
	if err != nil {
		return nil, err
	}

	projects := make(map[string]httputil.ProjectAccess)

	for _, permission := range userPermissions {
		if permission.ResourceType != model.ResourceNamespace {
			continue
		}
		pa, ok := projects[permission.ProjectID]
		if !ok {
			pa.ProjectID = permission.ProjectID
			pa.ProjectLabel = permission.ProjectLabel
		}
		pa.NamespacesAccesses = append(pa.NamespacesAccesses, httputil.NamespaceAccess{
			NamespaceID:    permission.ResourceID,
			NamespaceLabel: permission.Label,
			Access:         permission.CurrentAccessLevel,
		})
		projects[permission.ProjectID] = pa
	}

	ret := make([]httputil.ProjectAccess, 0)
	for _, project := range projects {
		ret = append(ret, project)
	}
	return ret, nil
}

func (s *Server) GetUserAccesses(ctx context.Context) ([]httputil.ProjectAccess, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Info("get user resource accesses")

	return extractAccessesFromDB(ctx, s.db, userID)
}

func (s *Server) SetUserAccesses(ctx context.Context, access kubeClientModel.UserGroupAccess) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Infof("Set user accesses to %s", access)

	err := s.db.Transactional(func(tx database.DB) error {
		if err := tx.SetUserAccesses(ctx, userID, access); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *Server) SetNamespaceAccess(ctx context.Context, id, targetUser string, accessLevel kubeClientModel.UserGroupAccess) error {
	ownerID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"owner_id":     ownerID,
		"target_user":  targetUser,
		"id":           id,
		"access_level": accessLevel,
	}).Debugf("set namespace access")

	err := s.db.Transactional(func(tx database.DB) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		ns, getErr := tx.NamespaceByID(ctx, ownerID, id)
		if getErr != nil {
			return getErr
		}

		if targetUserInfo.ID == ns.OwnerUserID {
			return errors.ErrSetOwnerAccess()
		}

		if chkErr := OwnerCheck(ctx, ns.Resource); chkErr != nil {
			return chkErr
		}

		if setErr := tx.SetNamespaceAccess(ctx, ns.Namespace, accessLevel, targetUserInfo.ID); setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}

func (s *Server) GetNamespaceAccesses(ctx context.Context, id string) (kubeClientModel.Namespace, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("get namespace accesses")

	ns, err := s.db.NamespaceByID(ctx, userID, id)
	if err != nil {
		return kubeClientModel.Namespace{}, err
	}
	err = s.db.NamespacePermissions(ctx, &ns)
	if err != nil {
		return ns.ToKube(), err
	}

	AddOwnerLogin(ctx, &ns.Resource, s.clients.User)
	AddUserLogins(ctx, ns.Permissions, s.clients.User)

	return ns.ToKube(), nil
}

func (s *Server) GetNamespaceAccess(ctx context.Context, id string) (httputil.NamespaceAccess, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("get namespace access")

	ns, err := s.db.NamespaceByID(ctx, userID, id)
	if err != nil {
		return httputil.NamespaceAccess{}, err
	}

	return httputil.NamespaceAccess{
		NamespaceID:    ns.ID,
		NamespaceLabel: ns.Label,
		Access:         ns.Permission.CurrentAccessLevel,
	}, nil
}

func (s *Server) DeleteNamespaceAccess(ctx context.Context, id string, targetUser string) error {
	ownerID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"owner_id":    ownerID,
		"id":          id,
		"target_user": targetUser,
	}).Debugf("delete namespace access")

	err := s.db.Transactional(func(tx database.DB) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		ns, getErr := tx.NamespaceByID(ctx, ownerID, id)
		if getErr != nil {
			return getErr
		}

		if chkErr := OwnerCheck(ctx, ns.Resource); chkErr != nil {
			return chkErr
		}

		if delErr := tx.DeleteNamespaceAccess(ctx, ns.Namespace, targetUserInfo.ID); delErr != nil {
			return delErr
		}

		return nil
	})

	return err
}
