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
	RenameVolume(ctx context.Context, id, newLabel string) error
	ResizeVolume(ctx context.Context, id string, newTariffID string) error
	GetVolume(ctx context.Context, id string) (model.VolumeWithPermissions, error)
	GetUserVolumes(ctx context.Context, filters ...string) ([]model.VolumeWithPermissions, error)
	GetAllVolumes(ctx context.Context, page, perPage int, filters ...string) ([]model.VolumeWithPermissions, error)
	DeleteVolume(ctx context.Context, id string) error
	DeleteAllUserVolumes(ctx context.Context) error
}

var StandardVolumeFilter = dao.VolumeFilter{
	NotDeleted: true,
}

func (s *Server) CreateVolume(ctx context.Context, req model.VolumeCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"tariff_id": req.TariffID,
		"id":        req.Label,
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

func (s *Server) GetVolume(ctx context.Context, id string) (model.VolumeWithPermissions, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("get volume")

	vol, err := s.db.VolumeByID(ctx, userID, id)
	if err != nil {
		return vol, err
	}

	AddOwnerLogin(ctx, &vol.Resource, s.clients.User)
	AddUserLogins(ctx, vol.Permissions, s.clients.User)

	return vol, nil
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

	vols, err := s.db.UserVolumes(ctx, userID, filter)
	if err != nil {
		return vols, err
	}

	for i := range vols {
		AddOwnerLogin(ctx, &vols[i].Resource, s.clients.User)
		AddUserLogins(ctx, vols[i].Permissions, s.clients.User)
	}

	return vols, nil
}

func (s *Server) GetAllVolumes(ctx context.Context, page, perPage int, filters ...string) ([]model.VolumeWithPermissions, error) {
	s.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
		"filters":  filters,
	}).Infof("get all volumes")

	var filter dao.VolumeFilter
	if len(filters) > 0 {
		filter = dao.ParseVolumeFilter()
	} else {
		filter = StandardVolumeFilter
	}
	filter.Limit = perPage
	filter.SetPage(page)

	vols, err := s.db.AllVolumes(ctx, filter)
	if err != nil {
		return vols, err
	}

	for i := range vols {
		AddOwnerLogin(ctx, &vols[i].Resource, s.clients.User)
		AddUserLogins(ctx, vols[i].Permissions, s.clients.User)
	}

	return vols, nil
}

func (s *Server) DeleteVolume(ctx context.Context, id string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("delete volume")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		vol, getErr := tx.VolumeByID(ctx, userID, id)
		if getErr != nil {
			return getErr
		}

		if chkErr := OwnerCheck(ctx, vol.Resource); chkErr != nil {
			return chkErr
		}

		if delErr := tx.DeleteVolume(ctx, &vol.Volume); delErr != nil {
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
		deletedVols, delErr := tx.DeleteAllVolumes(ctx, userID)
		if delErr != nil {
			return delErr
		}

		// TODO: actually delete it

		var resourceIDs []string
		for _, v := range deletedVols {
			resourceIDs = append(resourceIDs, v.ID)
		}
		if unsubErr := s.clients.Billing.MassiveUnsubscribe(ctx, resourceIDs); unsubErr != nil {
			return unsubErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) RenameVolume(ctx context.Context, id, newLabel string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
		"new_id":  newLabel,
	}).Infof("rename volume")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		vol, getErr := tx.VolumeByID(ctx, userID, id)
		if getErr != nil {
			return getErr
		}

		if chkErr := OwnerCheck(ctx, vol.Resource); chkErr != nil {
			return chkErr
		}

		if renErr := tx.RenameVolume(ctx, &vol.Volume, newLabel); renErr != nil {
			return renErr
		}

		// TODO: rename it actually

		if renErr := s.clients.Billing.Rename(ctx, vol.ID, newLabel); renErr != nil {
			return renErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) ResizeVolume(ctx context.Context, id string, newTariffID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"id":            id,
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
		vol, getErr := tx.VolumeByID(ctx, userID, id)
		if getErr != nil {
			return getErr
		}

		if chkErr := OwnerCheck(ctx, vol.Resource); chkErr != nil {
			return chkErr
		}

		vol.TariffID = &newTariff.ID
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
