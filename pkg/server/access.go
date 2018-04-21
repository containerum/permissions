package server

import (
	"context"

	"git.containerum.net/ch/auth/proto"
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
}

func (s *Server) GetUserAccesses(ctx context.Context, userID string) (*authProto.ResourcesAccess, error) {
	s.log.WithField("user_id", userID).Info("get user resource accesses")

	userPermissions, err := s.db.GetUserAccesses(ctx, userID)
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

func (s *Server) SetUserAccesses(ctx context.Context, userID string, access model.AccessLevel) error {
	s.log.WithField("user_id", userID).Infof("Set user accesses to %s", access)

	err := s.db.Transactional(func(tx *dao.DAO) error {
		if err := tx.SetUserAccesses(ctx, userID, access); err != nil {
			return err
		}

		// TODO: update accesses on auth

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
		targetUserID := targetUser // TODO: get from user manager

		ns, getErr := tx.NamespaceByLabel(ctx, ownerID, label)
		if getErr != nil {
			return getErr
		}

		if ns.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("namespace %s not owned by user", label)
		}

		if setErr := tx.SetNamespaceAccess(ctx, ns, accessLevel, targetUserID); setErr != nil {
			return setErr
		}

		// TODO: update auth

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
		targetUserID := targetUser // TODO: get from user manager

		vol, getErr := tx.VolumeByLabel(ctx, ownerID, label)
		if getErr != nil {
			return getErr
		}

		if vol.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("volume %s not owned by user", label)
		}

		if setErr := tx.SetVolumeAccess(ctx, vol, accessLevel, targetUserID); setErr != nil {
			return setErr
		}

		// TODO: update auth

		return nil
	})

	return err
}
