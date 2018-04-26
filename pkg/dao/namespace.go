package dao

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/pg"
	"github.com/sirupsen/logrus"
)

func (dao *DAO) NamespaceByID(ctx context.Context, id string) (ret model.Namespace, err error) {
	dao.log.WithField("id", id).Debugf("get namespace by id")

	err = dao.db.Model(&ret).
		Where("id = ?", id).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("namespace with id %s no exists", id)
	default:
		err = dao.handleError(err)

	}

	return
}

func (dao *DAO) NamespaceByLabel(ctx context.Context, userID, label string) (ret model.Namespace, err error) {
	dao.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("get namespace by user id and label")

	err = dao.db.Model(&ret).
		Column("\"?TableName\".*").
		Join("JOIN permissions").
		JoinOn("permissions.kind = ?", "Namespace").
		JoinOn("permissions.resource_id = \"?TableName\".id").
		Where("permissions.user_id = ?", userID).
		Where("\"?TableName\".label = ?", label).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("namespace %s not exists for user", label)
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) CreateNamespace(ctx context.Context, namespace *model.Namespace) error {
	dao.log.Debugf("create namespace %+v", namespace)

	_, err := dao.db.Model(namespace).
		OnConflict("(owner_user_id, label) DO UPDATE").
		Set("ram = ?ram").
		Set("cpu = ?cpu").
		Set("max_ext_services = ?max_ext_services").
		Set("max_int_services = ?max_int_services").
		Set("max_traffic = ?max_traffic").
		Returning("*").
		Insert()
	if err != nil {
		err = dao.handleError(err)
	}

	return err
}
