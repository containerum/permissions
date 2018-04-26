package server

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/dao"
	"git.containerum.net/ch/permissions/pkg/model"
	"git.containerum.net/ch/utils/httputil"
)

type NamespaceActions interface {
	AdminCreateNamespace(ctx context.Context, req model.NamespaceAdminCreateRequest) error
}

func (s *Server) AdminCreateNamespace(ctx context.Context, req model.NamespaceAdminCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithField("user_id", userID).Infof("admin create namespace %+v", req)

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns := model.Namespace{
			Resource: model.Resource{
				OwnerUserID: userID,
				Label:       req.Label,
			},
			CPU:            req.CPU,
			RAM:            req.Memory,
			MaxExtServices: req.MaxExtServices,
			MaxIntServices: req.MaxIntServices,
			MaxTraffic:     req.MaxTraffic,
		}

		if createErr := tx.CreateNamespace(ctx, &ns); createErr != nil {
			return createErr
		}

		// TODO: create in kube

		if updErr := updateUserAccesses(ctx, s.clients.Auth, s.db, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}
