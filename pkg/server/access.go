package server

import (
	"context"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
)

type AccessActions interface {
	GetUserAccesses(ctx context.Context, userID string) (*authProto.ResourcesAccess, error)
}

func (s *Server) GetUserAccesses(ctx context.Context, userID string) (*authProto.ResourcesAccess, error) {
	s.log.WithField("user_id", userID).Info("get user resource accesses")

	var userPermissions []struct {
		model.Permission `pg:",override"`

		Label string `sql:"label"`
	}
	err := s.db.Model(&userPermissions).
		Column("permissions.*").
		ColumnExpr("coalesce(ns.label, vol.label) AS label").
		Join("LEFT JOIN namespaces AS ns").JoinOn("permissions.resource_id = ns.id").JoinOn("permissions.resource_kind = 'namespace'").
		Join("LEFT JOIN volumes AS vol").JoinOn("permissions.resource_id = vol.id").JoinOn("permissions.resource_kind = 'volume'").
		Where("permissions.user_id = ?", userID).
		Where("label IS NOT NULL").
		Select()
	if err != nil {
		return nil, errors.ErrDatabase().Log(err, s.log)
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
