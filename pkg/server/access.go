package server

import (
	"context"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/permissions/pkg/clients"
	"git.containerum.net/ch/permissions/pkg/dao"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type AccessActions interface {
	GetUserAccesses(ctx context.Context) (*authProto.ResourcesAccess, error)
	SetUserAccesses(ctx context.Context, accessLevel model.AccessLevel) error
	GetNamespaceAccess(ctx context.Context, id string) (model.NamespaceWithPermissions, error)
	SetNamespaceAccess(ctx context.Context, id, targetUser string, accessLevel model.AccessLevel) error
	GetVolumeAccess(ctx context.Context, id string) (model.VolumeWithPermissions, error)
	SetVolumeAccess(ctx context.Context, id, targetUser string, accessLevel model.AccessLevel) error
	DeleteNamespaceAccess(ctx context.Context, id string, targetUser string) error
	DeleteVolumeAccess(ctx context.Context, id string, targetUser string) error
}

func extractAccessesFromDB(ctx context.Context, db *dao.DAO, userID string) (*authProto.ResourcesAccess, error) {
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

func updateUserAccesses(ctx context.Context, auth clients.AuthClient, db *dao.DAO, userID string) error {
	accesses, err := extractAccessesFromDB(ctx, db, userID)
	if err != nil {
		return err
	}

	return auth.UpdateUserAccess(ctx, userID, accesses)
}

func (s *Server) SetUserAccesses(ctx context.Context, access model.AccessLevel) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Infof("Set user accesses to %s", access)

	err := s.db.Transactional(func(tx *dao.DAO) error {
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

func (s *Server) SetNamespaceAccess(ctx context.Context, id, targetUser string, accessLevel model.AccessLevel) error {
	ownerID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"owner_id":     ownerID,
		"target_user":  targetUser,
		"id":           id,
		"access_level": accessLevel,
	}).Debugf("set namespace access")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		ns, getErr := tx.NamespaceByID(ctx, ownerID, id)
		if getErr != nil {
			return getErr
		}

		if ns.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("namespace %s not owned by user", id)
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

func (s *Server) GetNamespaceAccess(ctx context.Context, id string) (model.NamespaceWithPermissions, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("get namespace access")

	ns, err := s.db.NamespaceByID(ctx, userID, id)
	if err != nil {
		return model.NamespaceWithPermissions{}, err
	}
	err = s.db.NamespacePermissions(ctx, &ns)

	// TODO: maybe better method for get user login list by id list
	for i, perm := range ns.Permissions {
		info, err := s.clients.User.UserInfoByID(ctx, perm.UserID)
		if err != nil {
			return model.NamespaceWithPermissions{}, err
		}

		ns.Permissions[i].UserLogin = info.Login
	}

	return ns, err
}

func (s *Server) SetVolumeAccess(ctx context.Context, id, targetUser string, accessLevel model.AccessLevel) error {
	ownerID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"owner_id":     ownerID,
		"target_user":  targetUser,
		"id":           id,
		"access_level": accessLevel,
	}).Debugf("set volume access")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		vol, getErr := tx.VolumeByID(ctx, ownerID, id)
		if getErr != nil {
			return getErr
		}

		if vol.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("volume %s not owned by user", id)
		}

		if setErr := tx.SetVolumeAccess(ctx, vol.Volume, accessLevel, targetUserInfo.ID); setErr != nil {
			return setErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, targetUserInfo.ID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) GetVolumeAccess(ctx context.Context, id string) (model.VolumeWithPermissions, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("get volume access")

	vol, err := s.db.VolumeByID(ctx, userID, id)
	if err != nil {
		return model.VolumeWithPermissions{}, err
	}
	err = s.db.VolumePermissions(ctx, &vol)

	// TODO: maybe better method for get user login list by id list
	for i, perm := range vol.Permissions {
		info, err := s.clients.User.UserInfoByID(ctx, perm.UserID)
		if err != nil {
			return model.VolumeWithPermissions{}, err
		}

		vol.Permissions[i].UserLogin = info.Login
	}

	return vol, err
}

func (s *Server) DeleteNamespaceAccess(ctx context.Context, id string, targetUser string) error {
	ownerID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"owner_id":    ownerID,
		"id":          id,
		"target_user": targetUser,
	}).Debugf("delete namespace access")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		ns, getErr := tx.NamespaceByID(ctx, ownerID, id)
		if getErr != nil {
			return getErr
		}

		if ns.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("namespace %s not owned by user", id)
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

func (s *Server) DeleteVolumeAccess(ctx context.Context, id string, targetUser string) error {
	ownerID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"owner_id":    ownerID,
		"id":          id,
		"target_user": targetUser,
	}).Debugf("delete volume access")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		targetUserInfo, err := s.clients.User.UserInfoByLogin(ctx, targetUser)
		if err != nil {
			return err
		}

		vol, getErr := tx.VolumeByID(ctx, ownerID, id)
		if getErr != nil {
			return getErr
		}

		if vol.OwnerUserID != ownerID {
			return errors.ErrResourceNotOwned().AddDetailF("volume %s not owned by user", id)
		}

		if delErr := tx.DeleteVolumeAccess(ctx, vol.Volume, targetUserInfo.ID); delErr != nil {
			return delErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, targetUserInfo.ID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}
