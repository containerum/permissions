package server

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	billing "github.com/containerum/bill-external/models"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type NamespaceActions interface {
	CreateNamespace(ctx context.Context, projectID string, req model.NamespaceCreateRequest) error
	GetNamespace(ctx context.Context, projectID, id string) (kubeClientModel.Namespace, error)
	GetUserNamespaces(ctx context.Context, filters ...string) ([]kubeClientModel.Namespace, error)
	GetAllNamespaces(ctx context.Context, page, perPage int, filters ...string) ([]kubeClientModel.Namespace, error)
	AdminCreateNamespace(ctx context.Context, projectID string, req model.NamespaceAdminCreateRequest) error
	AdminResizeNamespace(ctx context.Context, projectID, id string, req model.NamespaceAdminResizeRequest) error
	RenameNamespace(ctx context.Context, projectID, id, newLabel string) error
	ResizeNamespace(ctx context.Context, projectID, id, newTariffID string) error
	DeleteNamespace(ctx context.Context, projectID, id string) error
	DeleteAllUserNamespaces(ctx context.Context) error
}

var StandardNamespaceFilter = database.NamespaceFilter{
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

func (s *Server) CreateNamespace(ctx context.Context, projectID string, req model.NamespaceCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithFields(logrus.Fields{
		"user_id":    userID,
		"tariff_id":  req.TariffID,
		"id":         req.Label,
		"project_id": projectID,
	}).Infof("create namespace")

	tariff, err := s.clients.Billing.GetNamespaceTariff(ctx, req.TariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(tariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	err = s.db.Transactional(func(tx database.DB) error {
		ns := model.NamespaceWithPermissions{
			Namespace: model.Namespace{
				Resource: model.Resource{
					OwnerUserID: userID,
					Label:       req.Label,
				},
				TariffID:       &req.TariffID,
				CPU:            tariff.CPULimit,
				RAM:            tariff.MemoryLimit,
				MaxExtServices: tariff.ExternalServices,
				MaxIntServices: tariff.InternalServices,
				MaxTraffic:     tariff.Traffic,
				ProjectID:      &projectID,
			},
		}

		if createErr := tx.CreateNamespace(ctx, &ns.Namespace); createErr != nil {
			return createErr
		}

		if createErr := s.clients.Kube.CreateNamespace(ctx, projectID, ns.ToKube()); createErr != nil {
			return createErr
		}

		if subErr := s.clients.Billing.Subscribe(ctx, billing.SubscribeTariffRequest{
			TariffID:      tariff.ID,
			ResourceType:  billing.Namespace,
			ResourceLabel: ns.Label,
			ResourceID:    ns.ID,
		}); subErr != nil {
			return subErr
		}

		return nil
	})

	return err
}

func (s *Server) GetNamespace(ctx context.Context, projectID, id string) (kubeClientModel.Namespace, error) {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithFields(logrus.Fields{
		"user_id":    userID,
		"id":         id,
		"project_id": projectID,
	}).Infof("get namespace")

	ns, err := s.db.NamespaceByID(ctx, userID, id)
	if err != nil {
		return kubeClientModel.Namespace{}, err
	}

	kubeNS := ns.ToKube()
	if kubeErr := NamespaceAddUsage(ctx, projectID, &kubeNS, s.clients.Kube); kubeErr != nil {
		s.log.WithError(kubeErr).Warn("NamespaceAddUsage failed")
		return kubeClientModel.Namespace{},
			errors.ErrResourceNotExists().AddDetailF("namespace %s not exists", id)
	}

	AddOwnerLogin(ctx, &ns.Resource, s.clients.User)
	AddUserLogins(ctx, ns.Permissions, s.clients.User)

	return kubeNS, nil
}

func (s *Server) GetUserNamespaces(ctx context.Context, filters ...string) ([]kubeClientModel.Namespace, error) {
	userID := httputil.MustGetUserID(ctx)

	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filters,
	}).Infof("get user namespaces")

	var filter database.NamespaceFilter
	if !IsAdminRole(ctx) {
		filter = StandardNamespaceFilter
	} else {
		filter = database.ParseNamespaceFilter(filters...)
	}

	namespaces, err := s.db.UserNamespaces(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	ret := make([]kubeClientModel.Namespace, 0)
	for _, namespace := range namespaces {
		AddOwnerLogin(ctx, &namespace.Resource, s.clients.User)
		kubeNS := namespace.ToKube()
		project := "64435c65-d553-4971-b00e-8ee5306f36d5"
		if namespace.ProjectID != nil {
			project = *namespace.ProjectID
		}
		kubeErr := NamespaceAddUsage(ctx, project, &kubeNS, s.clients.Kube)
		if kubeErr != nil {
			s.log.WithError(kubeErr).Warn("NamespaceAddUsage failed")
		}
		ret = append(ret, kubeNS)
	}

	return ret, nil
}

func (s *Server) GetAllNamespaces(ctx context.Context, page, perPage int, filters ...string) ([]kubeClientModel.Namespace, error) {
	s.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
		"filters":  filters,
	}).Infof("get all namespaces")

	var filter database.NamespaceFilter
	if len(filters) > 0 {
		filter = database.ParseNamespaceFilter(filters...)
	} else {
		filter = StandardNamespaceFilter
	}
	filter.Limit = perPage
	filter.SetPage(page)

	namespaces, err := s.db.AllNamespaces(ctx, filter)
	if err != nil {
		return nil, err
	}

	ret := make([]kubeClientModel.Namespace, 0)
	for _, namespace := range namespaces {
		AddOwnerLogin(ctx, &namespace.Resource, s.clients.User)
		kubeNS := (&model.NamespaceWithPermissions{Namespace: namespace}).ToKube()
		project := "64435c65-d553-4971-b00e-8ee5306f36d5"
		if namespace.ProjectID != nil {
			project = *namespace.ProjectID
		}
		kubeErr := NamespaceAddUsage(ctx, project, &kubeNS, s.clients.Kube)
		if kubeErr != nil {
			s.log.WithError(kubeErr).Warn("NamespaceAddUsage failed")
		}
		ret = append(ret, kubeNS)
	}

	return ret, nil
}

func (s *Server) AdminCreateNamespace(ctx context.Context, projectID string, req model.NamespaceAdminCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.
		WithField("user_id", userID).
		WithField("project_id", projectID).
		Infof("admin create namespace %+v", req)

	err := s.db.Transactional(func(tx database.DB) error {
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
				ProjectID:      &projectID,
			},
		}

		if createErr := tx.CreateNamespace(ctx, &ns.Namespace); createErr != nil {
			return createErr
		}

		if createErr := s.clients.Kube.CreateNamespace(ctx, projectID, ns.ToKube()); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (s *Server) AdminResizeNamespace(ctx context.Context, projectID, id string, req model.NamespaceAdminResizeRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.
		WithField("user_id", userID).
		WithField("id", id).
		WithField("project_id", projectID).
		Infof("admin resize namespace %+v", req)

	err := s.db.Transactional(func(tx database.DB) error {
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

		if ns.ProjectID != nil {
			projectID = *ns.ProjectID
		}

		nsWithUsage, getErr := s.clients.Kube.GetNamespace(ctx, projectID, ns.ID)
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

		if setErr := s.clients.Kube.SetNamespaceQuota(ctx, projectID, kubeNS); setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}

func (s *Server) RenameNamespace(ctx context.Context, projectID, id, newLabel string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":    userID,
		"id":         id,
		"new_id":     newLabel,
		"project_id": projectID,
	}).Infof("rename namespace")

	err := s.db.Transactional(func(tx database.DB) error {
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

		return nil
	})

	return err
}

func (s *Server) ResizeNamespace(ctx context.Context, projectID, id, newTariffID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"id":            id,
		"new_tariff_id": newTariffID,
		"project_id":    projectID,
	}).Infof("resize namespace")

	newTariff, err := s.clients.Billing.GetNamespaceTariff(ctx, newTariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(newTariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	err = s.db.Transactional(func(tx database.DB) error {
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

		nsWithUsage, getErr := s.clients.Kube.GetNamespace(ctx, projectID, ns.ID)
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

		if resizeErr := s.clients.Kube.SetNamespaceQuota(ctx, projectID, ns.ToKube()); resizeErr != nil {
			return resizeErr
		}

		if oldTariff.VolumeSize <= 0 && newTariff.VolumeSize > 0 {
			if createErr := s.clients.Volume.CreateVolume(ctx, projectID, ns.ID, DefaultVolumeName, newTariff.VolumeSize); createErr != nil {
				return createErr
			}
		}

		if resizeErr := s.clients.Billing.UpdateSubscription(ctx, ns.ID, newTariff.ID); resizeErr != nil {
			return resizeErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteNamespace(ctx context.Context, projectID, id string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":    userID,
		"id":         id,
		"project_id": projectID,
	}).Infof("delete namespace")

	err := s.db.Transactional(func(tx database.DB) error {
		ns, getErr := tx.NamespaceByID(ctx, userID, id)
		if getErr != nil {
			return getErr
		}

		if chkErr := OwnerCheck(ctx, ns.Resource); chkErr != nil {
			return chkErr
		}

		if delErr := tx.DeleteNamespace(ctx, &ns.Namespace); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Solutions.DeleteNamespaceSolutions(ctx, projectID, ns.ID); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Resource.DeleteNamespaceResources(ctx, projectID, ns.ID); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Volume.DeleteNamespaceVolumes(ctx, projectID, ns.ID); delErr != nil {
			return delErr
		}

		resourceIDs := []string{ns.ID}
		if unsubErr := s.clients.Billing.MassiveUnsubscribe(ctx, resourceIDs); unsubErr != nil {
			return unsubErr
		}

		if delErr := s.clients.Kube.DeleteNamespace(ctx, projectID, ns.ToKube()); delErr != nil {
			return delErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteAllUserNamespaces(ctx context.Context) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Infof("delete all user namespaces")

	err := s.db.Transactional(func(tx database.DB) error {
		deletedNamespaces, delErr := tx.DeleteAllUserNamespaces(ctx, userID)
		if delErr != nil {
			return delErr
		}

		var resourceIDs []string
		for _, v := range deletedNamespaces {
			resourceIDs = append(resourceIDs, v.ID)
		}

		if unsubErr := s.clients.Billing.MassiveUnsubscribe(ctx, resourceIDs); unsubErr != nil {
			return unsubErr
		}

		if delErr := s.clients.Solutions.DeleteUserSolutions(ctx); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Resource.DeleteAllUserNamespaces(ctx); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Kube.DeleteUserNamespaces(ctx, userID); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Volume.DeleteAllUserVolumes(ctx); delErr != nil {
			return delErr
		}

		return nil
	})

	return err
}
