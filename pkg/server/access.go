package server

import (
	"context"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/permissions/pkg/dao"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
)

type AccessActions interface {
	GetUserAccesses(ctx context.Context, userID string) (*authProto.ResourcesAccess, error)
	SetUserAccesses(ctx context.Context, userID string, accessLevel model.AccessLevel) error
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
		case model.ResourceNamespace:
			ret.Namespace = append(ret.Namespace, obj)
		case model.ResourceVolume:
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

	if err != nil {
		return errors.ErrDatabase().Log(err, s.log)
	}

	return nil
}
