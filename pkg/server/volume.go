package server

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/dao"
	"git.containerum.net/ch/permissions/pkg/model"
	billing "github.com/containerum/bill-external/models"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type VolumeActions interface {
	CreateVolume(ctx context.Context, req model.VolumeCreateRequest) error
}

func (s *Server) CreateVolume(ctx context.Context, req model.VolumeCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"tariff_id": req.TariffID,
		"label":     req.Label,
		"user_id":   userID,
	}).Infof("create volume")

	tariff, err := s.clients.Billing.GetVolumeTariff(ctx, req.TariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(tariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	true := true // so go don`t allows to take address of constant (needed in model.Volume) we use this

	err = s.db.Transactional(func(tx *dao.DAO) error {
		storage, getErr := tx.LeastUsedStorage(ctx, tariff.StorageLimit)
		if getErr != nil {
			return getErr
		}

		volume := model.Volume{
			Resource: model.Resource{
				TariffID:    &tariff.ID,
				OwnerUserID: userID,
				Label:       req.Label,
			},
			Active:      &true,
			Capacity:    tariff.StorageLimit,
			Replicas:    tariff.ReplicasLimit,
			NamespaceID: nil,       // because it persistent
			GlusterName: req.Label, // can be changed in future
			StorageID:   storage.ID,
		}

		if createErr := tx.CreateVolume(ctx, &volume); createErr != nil {
			return createErr
		}

		// TODO: create it actually

		if updErr := updateUserAccesses(ctx, s.clients.Auth, s.db, userID); updErr != nil {
			return updErr
		}

		subReq := billing.SubscribeTariffRequest{
			TariffID:      tariff.ID,
			ResourceType:  billing.Volume,
			ResourceLabel: volume.Label,
			ResourceID:    volume.ID,
		}
		if subErr := s.clients.Billing.Subscribe(ctx, subReq); subErr != nil {
			return subErr
		}

		return nil
	})

	return err
}
