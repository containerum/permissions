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
	RenameVolume(ctx context.Context, label, newLabel string) error
	GetVolume(ctx context.Context, label string) (model.VolumeWithPermissions, error)
	GetUserVolumes(ctx context.Context, filters ...string) ([]model.VolumeWithPermissions, error)
	GetAllVolumes(ctx context.Context, page, perPage int, filters ...string) ([]model.VolumeWithPermissions, error)
	DeleteVolume(ctx context.Context, label string) error
	DeleteAllUserVolumes(ctx context.Context) error
}

var StandardVolumeFilter = dao.VolumeFilter{
	NotDeleted: true,
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

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
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

func (s *Server) GetVolume(ctx context.Context, label string) (model.VolumeWithPermissions, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Infof("get volume")

	return s.db.VolumeByLabel(ctx, userID, label)
}

func (s *Server) GetUserVolumes(ctx context.Context, filters ...string) ([]model.VolumeWithPermissions, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filters,
	}).Infof("get user volumes")

	var filter dao.VolumeFilter
	if IsAdminRole(ctx) {
		filter = dao.ParseVolumeFilter(filters...)
	} else {
		filter = StandardVolumeFilter
	}

	return s.db.UserVolumes(ctx, userID, filter)
}

func (s *Server) GetAllVolumes(ctx context.Context, page, perPage int, filters ...string) ([]model.VolumeWithPermissions, error) {
	s.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
		"filters":  filters,
	}).Infof("get all volumes")

	filter := dao.ParseVolumeFilter()
	filter.Limit = perPage
	filter.SetPage(page)

	return s.db.AllVolumes(ctx, filter)
}

func (s *Server) DeleteVolume(ctx context.Context, label string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Infof("delete volume")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		vol := &model.Volume{Resource: model.Resource{OwnerUserID: userID, Label: label}}

		if delErr := tx.DeleteVolume(ctx, vol); delErr != nil {
			return delErr
		}

		// TODO: actually delete it

		if unsubErr := s.clients.Billing.Unsubscribe(ctx, vol.ID); unsubErr != nil {
			return unsubErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteAllUserVolumes(ctx context.Context) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Infof("delete all user volumes")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		_, delErr := tx.DeleteAllVolumes(ctx, userID)
		if delErr != nil {
			return delErr
		}

		// TODO: actually delete it

		// TODO: unsubscribe all (needed method)

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) RenameVolume(ctx context.Context, label, newLabel string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"label":     label,
		"new_label": newLabel,
	}).Infof("rename volume")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		vol := &model.Volume{Resource: model.Resource{OwnerUserID: userID, Label: label}}

		if renErr := tx.RenameVolume(ctx, vol, newLabel); renErr != nil {
			return renErr
		}

		// TODO: rename it actually

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) ResizeVolume(ctx context.Context, label string, newTariffID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"label":         label,
		"new_tariff_id": newTariffID,
	}).Infof("resize volume")

	newTariff, err := s.clients.Billing.GetVolumeTariff(ctx, newTariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(newTariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	err = s.db.Transactional(func(tx *dao.DAO) error {
		vol, getErr := tx.VolumeByLabel(ctx, userID, label)
		if getErr != nil {
			return getErr
		}

		vol.Replicas = newTariff.ReplicasLimit
		vol.Capacity = newTariff.StorageLimit

		if resizeErr := tx.ResizeVolume(ctx, vol.Volume); resizeErr != nil {
			return resizeErr
		}

		// TODO: resize it actually

		return nil
	})

	return err
}
