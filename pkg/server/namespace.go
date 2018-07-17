package server

import (
	"context"

	"git.containerum.net/ch/kube-api/pkg/kubeErrors"
	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	billing "github.com/containerum/bill-external/models"
	"github.com/containerum/cherry"
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
	AddGroupNamespace(ctx context.Context, namespace, groupID string) error
	SetGroupMemberNamespaceAccess(ctx context.Context, namespace, groupID string, req model.SetGroupMemberAccessRequest) error
	GetNamespaceGroups(ctx context.Context, projectID string) ([]kubeClientModel.UserGroup, error)
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
			},
		}

		if createErr := tx.CreateNamespace(ctx, &ns.Namespace); createErr != nil {
			return createErr
		}

		if createErr := s.clients.Kube.CreateNamespace(ctx, ns.ToKube()); createErr != nil {
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

	kubeNS := ns.ToKube()
	if kubeErr := NamespaceAddUsage(ctx, &kubeNS, s.clients.Kube); kubeErr != nil {
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
		kubeErr := NamespaceAddUsage(ctx, &kubeNS, s.clients.Kube)
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
		kubeErr := NamespaceAddUsage(ctx, &kubeNS, s.clients.Kube)
		if kubeErr != nil {
			s.log.WithError(kubeErr).Warn("NamespaceAddUsage failed")
		}
		ret = append(ret, kubeNS)
	}

	return ret, nil
}

func (s *Server) AdminCreateNamespace(ctx context.Context, req model.NamespaceAdminCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)

	s.log.
		WithField("user_id", userID).
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

		if resizeErr := s.clients.Kube.SetNamespaceQuota(ctx, ns.ToKube()); resizeErr != nil {
			return resizeErr
		}

		if oldTariff.VolumeSize <= 0 && newTariff.VolumeSize > 0 {
			if createErr := s.clients.Volume.CreateVolume(ctx, ns.ID, DefaultVolumeName, newTariff.VolumeSize); createErr != nil {
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

func (s *Server) DeleteNamespace(ctx context.Context, id string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
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

		if delErr := s.clients.Solutions.DeleteNamespaceSolutions(ctx, ns.ID); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Resource.DeleteNamespaceResources(ctx, ns.ID); delErr != nil {
			return delErr
		}

		if delErr := s.clients.Volume.DeleteNamespaceVolumes(ctx, ns.ID); delErr != nil {
			return delErr
		}

		resourceIDs := []string{ns.ID}
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

		// kube-api don`t have method to delete list of namespaces
		for _, ns := range deletedNamespaces {
			nsPerm := model.NamespaceWithPermissions{Namespace: ns}
			if delErr := s.clients.Kube.DeleteNamespace(ctx, nsPerm.ToKube()); delErr != nil {
				switch {
				case delErr == nil: //pass
				case cherry.Equals(delErr, kubeErrors.ErrResourceNotExist()):
					s.log.WithError(delErr).Warnf("namespace not found in kube")
				default:
					return delErr
				}
			}
		}

		if delErr := s.clients.Volume.DeleteAllUserVolumes(ctx); delErr != nil {
			return delErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (s *Server) AddGroupNamespace(ctx context.Context, namespace, groupID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"group_id":  groupID,
		"namespace": namespace,
	}).Info("add group")

	group, err := s.clients.User.Group(ctx, groupID)
	if err != nil {
		return err
	}

	var accessList []database.AccessListElement
	for _, v := range group.Members {
		accessList = append(accessList, database.AccessListElement{
			AccessLevel: UserGroupAccessToDBAccess(v.Access),
			ToUserID:    v.Username,
		})
	}

	err = s.db.Transactional(func(tx database.DB) error {
		ns, getErr := tx.NamespaceByID(ctx, userID, namespace)
		if getErr != nil {
			return getErr
		}

		return tx.SetNamespacesAccesses(ctx, []model.Namespace{ns.Namespace}, accessList)
	})

	return err
}

func (s *Server) SetGroupMemberNamespaceAccess(ctx context.Context, namespace, groupID string, req model.SetGroupMemberAccessRequest) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"namespace": namespace,
		"group":     groupID,
		"username":  req.Username,
		"user_id":   userID,
		"access":    req.AccessLevel,
	}).Infof("set group member access")

	err := s.db.Transactional(func(tx database.DB) error {
		ns, getErr := tx.NamespaceByID(ctx, userID, namespace)
		if getErr != nil {
			return getErr
		}

		user, getErr := s.clients.User.UserInfoByLogin(ctx, req.Username)
		if getErr != nil {
			return getErr
		}

		accesses := []database.AccessListElement{
			{ToUserID: user.ID, AccessLevel: UserGroupAccessToDBAccess(req.AccessLevel)},
		}
		if setErr := tx.SetNamespacesAccesses(ctx, []model.Namespace{ns.Namespace}, accesses); setErr != nil {
			return setErr
		}

		return updateUserAccesses(ctx, s.clients.Auth, s.db, user.ID)
	})

	return err
}

func (s *Server) GetNamespaceGroups(ctx context.Context, namespace string) ([]kubeClientModel.UserGroup, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"namespace": namespace,
		"user_id":   userID,
	}).Infof("get project groups")

	ns, err := s.db.NamespaceByID(ctx, userID, namespace)
	if err != nil {
		return nil, err
	}

	var groupIDs []string
	for _, v := range ns.Permissions {
		if v.GroupID != nil {
			groupIDs = append(groupIDs, *v.GroupID)
		}
	}

	groups, err := s.clients.User.GroupFullIDList(ctx, groupIDs...)
	if err != nil {
		return nil, err
	}

	return groups.Groups, nil
}
