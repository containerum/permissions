package server

import (
	"context"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/permissions/pkg/clients"
	"git.containerum.net/ch/permissions/pkg/dao"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/sirupsen/logrus"
)

type AccessActions interface {
	GetUserAccesses(ctx context.Context, userID string) (*authProto.ResourcesAccess, error)
	SetUserAccesses(ctx context.Context, userID string, accessLevel model.AccessLevel) error
	SetNamespaceAccess(ctx context.Context, ownerID, label, targetUser string, accessLevel model.AccessLevel) error
	SetVolumeAccess(ctx context.Context, ownerID, label, targetUser string, accessLevel model.AccessLevel) error
	DeleteNamespaceAccess(ctx context.Context, ownerID, label string, targetUser string) error
	DeleteVolumeAccess(ctx context.Context, ownerID, label string, targetUser string) error
}

func extractAccessesFromDB(ctx context.Context, db *dao.DAO, userID string) (*authProto.ResourcesAccess, error) {
	userPermissions, err := db.GetUserAccesses(ctx, userID)
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
		switch permission.ResourceKind {
		case "Namespace":
			ret.Namespace = append(ret.Namespace, obj)
		case "Volume":
			ret.Volume = append(ret.Volume, obj)
		}
	}

	return ret, nil
}

func (s *Server) GetUserAccesses(ctx context.Context, userID string) (*authProto.ResourcesAccess, error) {
	s.log.WithField("user_id", userID).Info("get user resource accesses")

	return extractAccessesFromDB(ctx, s.db, userID)
}

func updateUserAccesses(ctx context.Context, auth clients.AuthClient, db *dao.DAO, userID string) error {
	accesses, err := extractAccessesFromDB(ctx, db, userID)
	if err != nil {
		return err
	}

	return auth.UpdateUserAccess(ctx, userID, accesses)
}

func (s *Server) SetUserAccesses(ctx context.Context, userID string, access model.AccessLevel) error {
	s.log.WithField("user_id", userID).Infof("Set user accesses to %s", access)

	err := s.db.Transactional(func(tx *dao.DAO) error {
		if err := tx.SetUserAccesses(ctx, userID, access); err != nil {
			return err
		}

		if err := updateUserAccesses(ctx, s.clients.Auth, s.db, userID); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *Server) SetNamespaceAccess(ctx context.Context, ownerID, label, targetUser string, accessLevel model.AccessLevel) error {
	s.log.WithFields(logrus.Fields{
		"owner_id":     ownerID,
		"target_user":  targetUser,
		"label":        label,
		"access_level": accessLevel,
	}).Debugf("set namespace access")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		ns, getErr := tx.NamespaceByLabel(ctx, ownerID, label)
		if getErr != nil {
			return getErr
		}

		if ns.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("namespace %s not owned by user", label)
		}

		if setErr := tx.SetNamespaceAccess(ctx, ns.Namespace, accessLevel, targetUserInfo.ID); setErr != nil {
			return setErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, s.db, targetUserInfo.ID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) SetVolumeAccess(ctx context.Context, ownerID, label, targetUser string, accessLevel model.AccessLevel) error {
	s.log.WithFields(logrus.Fields{
		"owner_id":     ownerID,
		"target_user":  targetUser,
		"label":        label,
		"access_level": accessLevel,
	}).Debugf("set volume access")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		vol, getErr := tx.VolumeByLabel(ctx, ownerID, label)
		if getErr != nil {
			return getErr
		}

		if vol.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("volume %s not owned by user", label)
		}

		if setErr := tx.SetVolumeAccess(ctx, vol.Volume, accessLevel, targetUserInfo.ID); setErr != nil {
			return setErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, s.db, targetUserInfo.ID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteNamespaceAccess(ctx context.Context, ownerID, label string, targetUser string) error {
	s.log.WithFields(logrus.Fields{
		"owner_id":    ownerID,
		"label":       label,
		"target_user": targetUser,
	}).Debugf("delete namespace access")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		ns, getErr := tx.NamespaceByLabel(ctx, ownerID, label)
		if getErr != nil {
			return getErr
		}

		if ns.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("namespace %s not owned by user", label)
		}

		if delErr := tx.DeleteNamespaceAccess(ctx, ns.Namespace, targetUserInfo.ID); delErr != nil {
			return delErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, s.db, targetUserInfo.ID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteVolumeAccess(ctx context.Context, ownerID, label string, targetUser string) error {
	s.log.WithFields(logrus.Fields{
		"owner_id":    ownerID,
		"label":       label,
		"target_user": targetUser,
	}).Debugf("delete volume access")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		vol, getErr := tx.VolumeByLabel(ctx, ownerID, label)
		if getErr != nil {
			return getErr
		}

		if vol.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("volume %s not owned by user", label)
		}

		if delErr := tx.DeleteVolumeAccess(ctx, vol.Volume, targetUserInfo.ID); delErr != nil {
			return delErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, s.db, targetUserInfo.ID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}
