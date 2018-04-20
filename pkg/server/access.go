package server

import (
	"context"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/pg"
)

type AccessActions interface {
	GetUserAccesses(ctx context.Context, userID string) (*authProto.ResourcesAccess, error)
	SetUserAccesses(ctx context.Context, userID string, accessLevel model.AccessLevel) error
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

func (s *Server) SetUserAccesses(ctx context.Context, userID string, access model.AccessLevel) error {
	s.log.WithField("user_id", userID).Infof("Set user accesses to %s", access)

	err := s.handleTransactionError(s.db.RunInTransaction(func(tx *pg.Tx) error {
		nsIDsQuery := tx.Model((*model.Namespace)(nil)).Column("id").Where("owner_user_id = ?", userID)
		volIDsQuery := tx.Model((*model.Volume)(nil)).Column("id").Where("owner_user_id = ?", userID)

		// We can lower initial access lever, upper current access level (but not greater then initial) or set to initial
		_, err := s.db.Model((*model.Permission)(nil)).
			Where("resource_id IN (? UNION ALL ?)", nsIDsQuery, volIDsQuery).
			Set(`current_access_level = CASE WHEN current_access_level > ?0 THEN ?0
											WHEN current_access_level <= ?0 AND initial_access_level > ?0 THEN ?0
											ELSE initial_access_level`, access).
			Update()
		if err != nil {
			return err
		}

		// TODO: update accesses on auth

		return nil
	}))

	if err != nil {
		return errors.ErrDatabase().Log(err, s.log)
	}

	return nil
}
