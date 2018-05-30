package server

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/dao"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	billing "github.com/containerum/bill-external/models"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type NamespaceActions interface {
	CreateNamespace(ctx context.Context, req model.NamespaceCreateRequest) error
	GetNamespace(ctx context.Context, id string) (kubeClientModel.Namespace, error)
	GetUserNamespaces(ctx context.Context, filters ...string) ([]kubeClientModel.Namespace, error)
	GetAllNamespaces(ctx context.Context, page, perPage int, filters ...string) ([]kubeClientModel.Namespace, error)
	AdminCreateNamespace(ctx context.Context, req model.NamespaceAdminCreateRequest) error
	AdminResizeNamespace(ctx context.Context, id string, req model.NamespaceAdminResizeRequest) error
	RenameNamespace(ctx context.Context, id, newLabel string) error
	ResizeNamespace(ctx context.Context, id, newTariffID string) error
	DeleteNamespace(ctx context.Context, id string) error
	DeleteAllUserNamespaces(ctx context.Context) error
}

var StandardNamespaceFilter = dao.NamespaceFilter{
	NotDeleted: true,
}

func checkResizeNSQuota(nsWithUsage, kubeNS kubeClientModel.Namespace) error {
	if nsWithUsage.Resources.Used.Memory > kubeNS.Resources.Hard.Memory ||
		nsWithUsage.Resources.Used.CPU > kubeNS.Resources.Hard.CPU {
		return errors.ErrQuotaExceeded().AddDetailF("exceeded %d CPU and %d MiB memory",
			nsWithUsage.Resources.Used.CPU-kubeNS.Resources.Hard.CPU,
			nsWithUsage.Resources.Used.Memory-kubeNS.Resources.Hard.Memory)
	}
	return nil
}

func (s *Server) CreateNamespace(ctx context.Context, req model.NamespaceCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"tariff_id": req.TariffID,
		"id":        req.Label,
	}).Infof("create namespace")

	tariff, err := s.clients.Billing.GetNamespaceTariff(ctx, req.TariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(tariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	err = s.db.Transactional(func(tx *dao.DAO) error {
		ns := model.NamespaceWithPermissions{
			Namespace: model.Namespace{
				Resource: model.Resource{
					OwnerUserID: userID,
					Label:       req.Label,
					TariffID:    &req.TariffID,
				},
				CPU:            tariff.CPULimit,
				RAM:            tariff.MemoryLimit,
				MaxExtServices: tariff.ExternalServices,
				MaxIntServices: tariff.InternalServices,
				MaxTraffic:     tariff.Traffic,
			},
		}

		if createErr := tx.CreateNamespace(ctx, &ns.Namespace); createErr != nil {
			return createErr
		}

		if tariff.VolumeSize > 0 {
			storage, getErr := tx.LeastUsedStorage(ctx, tariff.VolumeSize)
			if getErr != nil {
				return getErr
			}

			vol := model.Volume{
				Resource: model.Resource{
					OwnerUserID: userID,
					Label:       NamespaceVolumeGlusterLabel(ns.Label),
				},
				Capacity:    tariff.VolumeSize,
				NamespaceID: ns.ID,
				GlusterName: VolumeGlusterName(ns.Label, userID),
				StorageID:   storage.ID,
			}

			createErr := tx.CreateVolume(ctx, &vol)
			if createErr != nil {
				return createErr
			}

			// TODO: create it actually
		}

		if createErr := s.clients.Kube.CreateNamespace(ctx, ns.ToKube()); createErr != nil {
			return createErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) GetNamespace(ctx context.Context, id string) (kubeClientModel.Namespace, error) {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("get namespace")

	ns, err := s.db.NamespaceByID(ctx, userID, id)
	if err != nil {
		return kubeClientModel.Namespace{}, err
	}

	err = s.db.NamespaceVolumes(ctx, &ns.Namespace)
	if err != nil {
		return ns.ToKube(), err
	}

	AddOwnerLogin(ctx, &ns.Resource, s.clients.User)
	AddUserLogins(ctx, ns.Permissions, s.clients.User)

	kubeNS := ns.ToKube()
	NamespaceAddUsage(ctx, &kubeNS, s.clients.Kube)

	return kubeNS, nil
}

func (s *Server) GetUserNamespaces(ctx context.Context, filters ...string) ([]kubeClientModel.Namespace, error) {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filters,
	}).Infof("get user namespaces")

	var filter dao.NamespaceFilter
	if !IsAdminRole(ctx) {
		filter = StandardNamespaceFilter
	} else {
		filter = dao.ParseNamespaceFilter(filters...)
	}

	namespaces, err := s.db.UserNamespaces(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	ret := make([]kubeClientModel.Namespace, len(namespaces))
	for i := range namespaces {
		AddOwnerLogin(ctx, &namespaces[i].Resource, s.clients.User)
		AddUserLogins(ctx, namespaces[i].Permissions, s.clients.User)
		ret[i] = namespaces[i].ToKube()
		NamespaceAddUsage(ctx, &ret[i], s.clients.Kube)
	}

	return ret, nil
}

func (s *Server) GetAllNamespaces(ctx context.Context, page, perPage int, filters ...string) ([]kubeClientModel.Namespace, error) {
	s.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
		"filters":  filters,
	}).Infof("get all namespaces")

	var filter dao.NamespaceFilter
	if len(filters) > 0 {
		filter = dao.ParseNamespaceFilter(filters...)
	} else {
		filter = StandardNamespaceFilter
	}
	filter.Limit = perPage
	filter.SetPage(page)

	namespaces, err := s.db.AllNamespaces(ctx, filter)
	if err != nil {
		return nil, err
	}

	ret := make([]kubeClientModel.Namespace, len(namespaces))
	for i := range namespaces {
		AddOwnerLogin(ctx, &namespaces[i].Resource, s.clients.User)
		ret[i] = (&model.NamespaceWithPermissions{Namespace: namespaces[i]}).ToKube()
		NamespaceAddUsage(ctx, &ret[i], s.clients.Kube)
	}

	return ret, nil
}

func (s *Server) AdminCreateNamespace(ctx context.Context, req model.NamespaceAdminCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.
		WithField("user_id", userID).
		Infof("admin create namespace %+v", req)

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns := model.NamespaceWithPermissions{
			Namespace: model.Namespace{
				Resource: model.Resource{
					OwnerUserID: userID,
					Label:       req.Label,
				},
				CPU:            req.CPU,
				RAM:            req.Memory,
				MaxExtServices: req.MaxExtServices,
				MaxIntServices: req.MaxIntServices,
				MaxTraffic:     req.MaxTraffic,
			},
		}

		if createErr := tx.CreateNamespace(ctx, &ns.Namespace); createErr != nil {
			return createErr
		}

		if createErr := s.clients.Kube.CreateNamespace(ctx, ns.ToKube()); createErr != nil {
			return createErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) AdminResizeNamespace(ctx context.Context, id string, req model.NamespaceAdminResizeRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.
		WithField("user_id", userID).
		WithField("id", id).
		Infof("admin resize namespace %+v", req)

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns, getErr := tx.NamespaceByID(ctx, userID, id)
		if getErr != nil {
			return getErr
		}

		if req.CPU != nil {
			ns.CPU = *req.CPU
		}
		if req.Memory != nil {
			ns.RAM = *req.Memory
		}
		if req.MaxExtServices != nil {
			ns.MaxExtServices = *req.MaxExtServices
		}
		if req.MaxIntServices != nil {
			ns.MaxIntServices = *req.MaxIntServices
		}
		if req.MaxTraffic != nil {
			ns.MaxTraffic = *req.MaxTraffic
		}

		nsWithUsage, getErr := s.clients.Kube.GetNamespace(ctx, ns.ID)
		if getErr != nil {
			return getErr
		}

		kubeNS := ns.ToKube()

		if chkErr := checkResizeNSQuota(nsWithUsage, kubeNS); chkErr != nil {
			return chkErr
		}

		if setErr := tx.ResizeNamespace(ctx, ns.Namespace); setErr != nil {
			return setErr
		}

		if setErr := s.clients.Kube.SetNamespaceQuota(ctx, kubeNS); setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}

func (s *Server) RenameNamespace(ctx context.Context, id, newLabel string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
		"new_id":  newLabel,
	}).Infof("rename namespace")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns, getErr := tx.NamespaceByID(ctx, userID, id)
		if getErr != nil {
			return getErr
		}

		if chkErr := OwnerCheck(ctx, ns.Resource); chkErr != nil {
			return chkErr
		}

		if renameErr := tx.RenameNamespace(ctx, &ns.Namespace, newLabel); renameErr != nil {
			return renameErr
		}

		if renameErr := s.clients.Billing.Rename(ctx, ns.ID, newLabel); renameErr != nil {
			return renameErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) ResizeNamespace(ctx context.Context, id, newTariffID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"id":            id,
		"new_tariff_id": newTariffID,
	}).Infof("resize namespace")

	newTariff, err := s.clients.Billing.GetNamespaceTariff(ctx, newTariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(newTariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	err = s.db.Transactional(func(tx *dao.DAO) error {
		ns, getErr := tx.NamespaceByID(ctx, userID, id)
		if getErr != nil {
			return getErr
		}

		if chkErr := OwnerCheck(ctx, ns.Resource); chkErr != nil {
			return chkErr
		}

		var oldTariff billing.NamespaceTariff
		if ns.TariffID != nil {
			oldTariff, getErr = s.clients.Billing.GetNamespaceTariff(ctx, *ns.TariffID)
			if getErr != nil {
				return getErr
			}
		}

		ns.TariffID = &newTariff.ID
		ns.MaxIntServices = newTariff.ExternalServices
		ns.MaxIntServices = newTariff.InternalServices
		ns.MaxTraffic = newTariff.Traffic
		ns.CPU = newTariff.CPULimit
		ns.RAM = newTariff.MemoryLimit

		nsWithUsage, getErr := s.clients.Kube.GetNamespace(ctx, ns.ID)
		if getErr != nil {
			return getErr
		}

		kubeNS := ns.ToKube()

		if chkErr := checkResizeNSQuota(nsWithUsage, kubeNS); chkErr != nil {
			return chkErr
		}

		if resizeErr := tx.ResizeNamespace(ctx, ns.Namespace); resizeErr != nil {
			return resizeErr
		}

		if oldTariff.VolumeSize == 0 && newTariff.VolumeSize > 0 {
			storage, getErr := tx.LeastUsedStorage(ctx, newTariff.VolumeSize)
			if getErr != nil {
				return getErr
			}

			vol := model.Volume{
				Resource: model.Resource{
					OwnerUserID: userID,
					Label:       NamespaceVolumeGlusterLabel(ns.Label),
				},
				Capacity:    newTariff.VolumeSize,
				NamespaceID: ns.ID,
				GlusterName: VolumeGlusterName(ns.Label, userID),
				StorageID:   storage.ID,
			}

			createErr := tx.CreateVolume(ctx, &vol)
			if createErr != nil {
				return createErr
			}
		}

		if oldTariff.VolumeSize > 0 && newTariff.VolumeSize == 0 {
			_, delErr := tx.DeleteNamespaceVolumes(ctx, ns.Namespace)
			if delErr != nil {
				return delErr
			}
		}

		if resizeErr := s.clients.Kube.SetNamespaceQuota(ctx, ns.ToKube()); resizeErr != nil {
			return resizeErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteNamespace(ctx context.Context, id string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("delete namespace")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns, getErr := tx.NamespaceByID(ctx, userID, id)
		if getErr != nil {
			return getErr
		}

		if chkErr := OwnerCheck(ctx, ns.Resource); chkErr != nil {
			return chkErr
		}

		deletedVols, delErr := tx.DeleteNamespaceVolumes(ctx, ns.Namespace)
		if delErr != nil {
			return delErr
		}

		if delErr := tx.DeleteNamespace(ctx, &ns.Namespace); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Resource.DeleteNamespaceResources(ctx, ns.ID); delErr != nil {
			return delErr
		}

		resourceIDs := []string{ns.ID}
		for _, v := range deletedVols {
			resourceIDs = append(resourceIDs, v.ID)
		}
		if unsubErr := s.clients.Billing.MassiveUnsubscribe(ctx, resourceIDs); unsubErr != nil {
			return unsubErr
		}

		if delErr := s.clients.Kube.DeleteNamespace(ctx, ns.ToKube()); delErr != nil {
			return delErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteAllUserNamespaces(ctx context.Context) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Infof("delete all user namespaces")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		deletedVols, delErr := tx.DeleteAllUserNamespaceVolumes(ctx, userID)
		if delErr != nil {
			return delErr
		}

		deletedNamespaces, delErr := tx.DeleteAllUserNamespaces(ctx, userID)
		if delErr != nil {
			return delErr
		}

		var resourceIDs []string
		for _, v := range deletedNamespaces {
			resourceIDs = append(resourceIDs, v.ID)
		}
		for _, v := range deletedVols {
			resourceIDs = append(resourceIDs, v.ID)
		}

		if unsubErr := s.clients.Billing.MassiveUnsubscribe(ctx, resourceIDs); unsubErr != nil {
			return unsubErr
		}

		if delErr := s.clients.Resource.DeleteAllUserNamespaces(ctx); delErr != nil {
			return delErr
		}

		// kube-api don`t have method to delete list of namespaces
		for _, ns := range deletedNamespaces {
			nsPerm := model.NamespaceWithPermissions{Namespace: ns}
			if delErr := s.clients.Kube.DeleteNamespace(ctx, nsPerm.ToKube()); delErr != nil {
				return delErr
			}
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}
