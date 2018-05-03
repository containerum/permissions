package dao

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
)

func (dao *DAO) VolumeByID(ctx context.Context, id string) (ret model.Volume, err error) {
	dao.log.WithField("id", id).Debugf("get volume by id")

	err = dao.db.Model(&ret).
		Where("id = ?", id).
		Where("NOT deleted").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("volume with id %s no exists", id)
	default:
		err = dao.handleError(err)

	}

	return
}

func (dao *DAO) VolumeByLabel(ctx context.Context, userID, label string) (ret model.VolumeWithPermissions, err error) {
	dao.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("get volume by user id and label")

	err = dao.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("permissions.*").
		Join("JOIN permissions").
		JoinOn("permissions.kind = ?", "Volume").
		JoinOn("permissions.resource_id = ?TableAlias.id").
		Where("permissions.user_id = ?", userID).
		Where("?TableAlias.label = ?", label).
		Where("NOT ?TableAlias.deleted").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("namespace %s not exists for user", label)
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) VolumePermissions(ctx context.Context, vol *model.VolumeWithPermissions) (err error) {
	dao.log.WithFields(logrus.Fields{
		"owner_user_id": vol.OwnerUserID,
		"label":         vol.Label,
	}).Debugf("get volume permissions")

	err = dao.db.Model(vol).
		Column("Permissions").
		Relation("Permissions", func(q *orm.Query) (*orm.Query, error) {
			return q.Where("initial_access_level != ?", model.AccessOwner), nil
		}).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = nil
	default:
		err = dao.handleError(err)
	}

	return
}
