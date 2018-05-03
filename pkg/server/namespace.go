package server

import (
	"context"
	"time"

	kubeAPIModel "git.containerum.net/ch/kube-api/pkg/model"
	kubeClientModel "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/permissions/pkg/dao"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type NamespaceActions interface {
	AdminCreateNamespace(ctx context.Context, req model.NamespaceAdminCreateRequest) error
	AdminResizeNamespace(ctx context.Context, label string, req model.NamespaceAdminResizeRequest) error
	DeleteNamespace(ctx context.Context, label string) error
	DeleteAllUserNamespaces(ctx context.Context) error
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

		createdAt := ns.CreateTime.Format(time.RFC3339)
		maxExtServices := uint(ns.MaxExtServices)
		maxIntServices := uint(ns.MaxIntServices)
		maxTraffic := uint(ns.MaxTraffic)
		nsKube := kubeAPIModel.NamespaceWithOwner{
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

		if createErr := s.clients.Kube.CreateNamespace(ctx, nsKube); createErr != nil {
			return createErr
		}

		if updErr := updateUserAccesses(ctx, s.clients.Auth, s.db, userID); updErr != nil {
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

		if setErr := tx.ResizeNamespace(ctx, ns); setErr != nil {
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

func (s *Server) DeleteNamespace(ctx context.Context, label string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Infof("delete namespace")

	err := s.db.Transactional(func(tx *dao.DAO) error {
		ns := &model.Namespace{Resource: model.Resource{OwnerUserID: userID, Label: label}}

		if delErr := tx.DeleteNamespace(ctx, ns); delErr != nil {
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
