package server

import (
	"context"
	"fmt"
	"time"

	kubeAPIModel "git.containerum.net/ch/kube-api/pkg/model"
	kubeClientModel "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/permissions/pkg/dao"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type NamespaceActions interface {
	CreateNamespace(ctx context.Context, req model.NamespaceCreateRequest) error
	GetNamespace(ctx context.Context, label string) (model.NamespaceWithPermissions, error)
	GetUserNamespaces(ctx context.Context, filters ...string) ([]model.NamespaceWithPermissions, error)
	GetAllNamespaces(ctx context.Context, page, perPage int, filters ...string) ([]model.NamespaceWithPermissions, error)
	AdminCreateNamespace(ctx context.Context, req model.NamespaceAdminCreateRequest) error
	AdminResizeNamespace(ctx context.Context, label string, req model.NamespaceAdminResizeRequest) error
	RenameNamespace(ctx context.Context, label, newLabel string) error
	DeleteNamespace(ctx context.Context, label string) error
	DeleteAllUserNamespaces(ctx context.Context) error
}

var StandardNamespaceFilter = dao.NamespaceFilter{
	NotDeleted: true,
}

func kubeNS(ns model.Namespace) kubeAPIModel.NamespaceWithOwner {
	createdAt := ns.CreateTime.Format(time.RFC3339)
	maxExtServices := uint(ns.MaxExtServices)
	maxIntServices := uint(ns.MaxIntServices)
	maxTraffic := uint(ns.MaxTraffic)
	return kubeAPIModel.NamespaceWithOwner{
		Namespace: kubeClientModel.Namespace{
			CreatedAt:     &createdAt,
			Label:         ns.Label,
			Access:        string(model.AccessOwner),
			MaxExtService: &maxExtServices,
			MaxIntService: &maxIntServices,
			MaxTraffic:    &maxTraffic,
			Resources: kubeClientModel.Resources{
				Hard: kubeClientModel.Resource{
					CPU:    uint(ns.CPU),
					Memory: uint(ns.RAM),
				},
			},
		},
		Name:   ns.ID,
		Owner:  ns.OwnerUserID,
		Access: string(model.AccessOwner),
	}
}

func (s *Server) CreateNamespace(ctx context.Context, req model.NamespaceCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"tariff_id": req.TariffID,
		"label":     req.Label,
	}).Infof("create namespace")

	tariff, err := s.clients.Billing.GetNamespaceTariff(ctx, req.TariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(tariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	err = s.db.Transactional(func(tx *dao.DAO) error {
		ns := model.Namespace{
			Resource: model.Resource{
				OwnerUserID: userID,
				Label:       req.Label,
			},
			CPU:            tariff.CPULimit,
			RAM:            tariff.MemoryLimit,
			MaxExtServices: tariff.ExternalServices,
			MaxIntServices: tariff.InternalServices,
			MaxTraffic:     tariff.Traffic,
		}

		if createErr := tx.CreateNamespace(ctx, &ns); createErr != nil {
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
					Label:       fmt.Sprintf("%s-volume", ns.Label),
				},
				Capacity:    tariff.VolumeSize,
				Replicas:    2,
				NamespaceID: &ns.ID,
				GlusterName: VolumeGlusterName(ns.Label, userID),
				StorageID:   storage.ID,
			}
			vol.Active = new(bool)

			createErr := tx.CreateVolume(ctx, &vol)
			if createErr != nil {
				return createErr
			}

			// TODO: create it actually
		}

		if createErr := s.clients.Kube.CreateNamespace(ctx, kubeNS(ns)); createErr != nil {
			return createErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) GetNamespace(ctx context.Context, label string) (model.NamespaceWithPermissions, error) {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Infof("get namespace")

	ns, err := s.db.NamespaceByLabel(ctx, userID, label)
	if err != nil {
		return model.NamespaceWithPermissions{}, err
	}

	err = s.db.NamespaceVolumes(ctx, &ns.Namespace)

	return ns, err
}

func (s *Server) GetUserNamespaces(ctx context.Context, filters ...string) ([]model.NamespaceWithPermissions, error) {
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

	return s.db.UserNamespaces(ctx, userID, filter)
}

func (s *Server) GetAllNamespaces(ctx context.Context, page, perPage int, filters ...string) ([]model.NamespaceWithPermissions, error) {
	s.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
		"filters":  filters,
	}).Infof("get all namespaces")

	filter := dao.ParseNamespaceFilter(filters...)
	filter.Limit = perPage
	filter.SetPage(page)

	return s.db.AllNamespaces(ctx, filter)
}

func (s *Server) AdminCreateNamespace(ctx context.Context, req model.NamespaceAdminCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.
		WithField("user_id", userID).
		Infof("admin create namespace %+v", req)

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

		if createErr := s.clients.Kube.CreateNamespace(ctx, kubeNS(ns)); createErr != nil {
			return createErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) AdminResizeNamespace(ctx context.Context, label string, req model.NamespaceAdminResizeRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.
		WithField("user_id", userID).
		WithField("label", label).
		Infof("admin resize namespace %+v", req)

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns, getErr := tx.NamespaceByLabel(ctx, userID, label)
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

		if setErr := tx.ResizeNamespace(ctx, ns.Namespace); setErr != nil {
			return setErr
		}

		kubeNS := kubeAPIModel.NamespaceWithOwner{
			Namespace: kubeClientModel.Namespace{
				Resources: kubeClientModel.Resources{
					Hard: kubeClientModel.Resource{
						CPU:    uint(ns.CPU),
						Memory: uint(ns.RAM),
					},
				},
			},
			Name:  ns.ID,
			Owner: ns.OwnerUserID,
		}
		if setErr := s.clients.Kube.SetNamespaceQuota(ctx, kubeNS); setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}

func (s *Server) RenameNamespace(ctx context.Context, label, newLabel string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"label":     label,
		"new_label": newLabel,
	}).Infof("rename namespace")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns, getErr := tx.NamespaceByLabel(ctx, userID, label)
		if getErr != nil {
			return getErr
		}

		if renameErr := tx.RenameNamespace(ctx, &ns.Namespace, newLabel); renameErr != nil {
			return renameErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteNamespace(ctx context.Context, label string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Infof("delete namespace")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns, getErr := tx.NamespaceByLabel(ctx, userID, label)
		if getErr != nil {
			return getErr
		}

		if _, delErr := tx.DeleteNamespaceVolumes(ctx, ns.Namespace); delErr != nil {
			return delErr
		}

		if delErr := tx.DeleteNamespace(ctx, &ns.Namespace); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Kube.DeleteNamespace(ctx, kubeAPIModel.NamespaceWithOwner{Name: ns.ID}); delErr != nil {
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
		deletedNamespaces, delErr := tx.DeleteAllUserNamespaces(ctx, userID)
		if delErr != nil {
			return delErr
		}

		if _, delErr := tx.DeleteAllUserNamespaceVolumes(ctx, userID); delErr != nil {
			return delErr
		}

		// kube-api don`t have method to delete list of namespaces
		for _, ns := range deletedNamespaces {
			if delErr := s.clients.Kube.DeleteNamespace(ctx, kubeAPIModel.NamespaceWithOwner{Name: ns.ID}); delErr != nil {
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
