package server

import (
	"context"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/permissions/pkg/clients"
	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type AccessActions interface {
	GetUserAccesses(ctx context.Context) (*authProto.ResourcesAccess, error)
	SetUserAccesses(ctx context.Context, accessLevel kubeClientModel.AccessLevel) error
	GetNamespaceAccess(ctx context.Context, id string) (kubeClientModel.Namespace, error)
	SetNamespaceAccess(ctx context.Context, id, targetUser string, accessLevel kubeClientModel.AccessLevel) error
	DeleteNamespaceAccess(ctx context.Context, id string, targetUser string) error
}

func extractAccessesFromDB(ctx context.Context, db database.DB, userID string) (*authProto.ResourcesAccess, error) {
	userPermissions, err := db.UserAccesses(ctx, userID)
	if err != nil {
		return nil, err
	}

	ret := &authProto.ResourcesAccess{
		Namespace: make([]*authProto.AccessObject, 0),
		Volume:    make([]*authProto.AccessObject, 0),
	}
	for _, permission := range userPermissions {
		obj := &authProto.AccessObject{
			Id:     permission.ResourceID,
			Access: string(permission.CurrentAccessLevel),
			Label:  permission.Label,
		}
		switch permission.ResourceType {
		case model.ResourceNamespace:
			ret.Namespace = append(ret.Namespace, obj)
		case model.ResourceVolume:
			ret.Volume = append(ret.Volume, obj)
		}
	}

	return ret, nil
}

func (s *Server) GetUserAccesses(ctx context.Context) (*authProto.ResourcesAccess, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Info("get user resource accesses")

	return extractAccessesFromDB(ctx, s.db, userID)
}

func updateUserAccesses(ctx context.Context, auth clients.AuthClient, db database.DB, userID string) error {
	accesses, err := extractAccessesFromDB(ctx, db, userID)
	if err != nil {
		return err
	}

	return auth.UpdateUserAccess(ctx, userID, accesses)
}

func (s *Server) SetUserAccesses(ctx context.Context, access kubeClientModel.AccessLevel) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Infof("Set user accesses to %s", access)

	err := s.db.Transactional(func(tx database.DB) error {
		if err := tx.SetUserAccesses(ctx, userID, access); err != nil {
			return err
		}

		if err := updateUserAccesses(ctx, s.clients.Auth, tx, userID); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *Server) SetNamespaceAccess(ctx context.Context, id, targetUser string, accessLevel kubeClientModel.AccessLevel) error {
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

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, targetUserInfo.ID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) GetNamespaceAccess(ctx context.Context, id string) (kubeClientModel.Namespace, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("get namespace access")

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

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, targetUserInfo.ID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}
