package dao

import (
	"context"
	"reflect"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
)

type VolumeFilter struct {
	orm.Pager
	NotDeleted    bool `filter:"not_deleted"`
	Deleted       bool `filter:"deleted"`
	NotLimited    bool `filter:"not_limited"`
	Limited       bool `filter:"limited"`
	Owned         bool `filter:"owner"`
	NotOwned      bool `filter:"not_owner"`
	Persistent    bool `filter:"persistent"`
	NotPersistent bool `filter:"not_persistent"`
}

var volFilterCache = make(map[string]int)

func init() {
	t := reflect.TypeOf(NamespaceFilter{})
	for i := 0; i < t.NumField(); i++ {
		tag, ok := t.Field(i).Tag.Lookup("filter")
		if !ok {
			continue
		}
		volFilterCache[tag] = i
	}
}

func ParseVolumeFilter(filters ...string) VolumeFilter {
	var ret VolumeFilter
	v := reflect.ValueOf(&ret)
	for _, filter := range filters {
		if field, ok := volFilterCache[filter]; ok {
			v.Field(field).SetBool(true)
		}
	}
	return ret
}

func (f *VolumeFilter) Filter(q *orm.Query) (*orm.Query, error) {
	if f.NotDeleted {
		q = q.Where("NOT ?TableAlias.deleted")
	}
	if f.Deleted {
		q = q.Where("?TableAlias.deleted")
	}
	if f.NotLimited {
		q = q.Where("permissions.initial_access_level = permissions.current_access_level")
	}
	if f.Limited {
		q = q.Where("permissions.initial_access_level != permissions.initial_access_level")
	}
	if f.Owned {
		q = q.Where("permissions.initial_access_level = ?", model.AccessOwner)
	}
	if f.NotOwned {
		q = q.Where("permissions.initial_access_level != ?", model.AccessOwner)
	}
	if f.Persistent {
		q = q.Where("?TableAlias.namespace_id IS NULL")
	}
	if f.NotPersistent {
		q = q.Where("?TableAlias.namespace_id IS NOT NULL")
	}

	return q.Apply(f.Paginate), nil
}

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

func (dao *DAO) UserVolumes(ctx context.Context, userID string, filter VolumeFilter) (ret []model.VolumeWithPermissions, err error) {
	dao.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filter,
	}).Debugf("get user volumes")

	err = dao.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("permissions.*").
		Join("JOIN permissions").
		JoinOn("permissions.kind = ?", "Volume").
		JoinOn("permissions.resource_id = ?TableAlias.id").
		Where("permissions.user_id = ?", userID).
		Apply(filter.Filter).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("user has no volumes")
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) AllVolumes(ctx context.Context, filter VolumeFilter) (ret []model.VolumeWithPermissions, err error) {
	dao.log.WithField("fields", filter).Debugf("get all volumes")

	err = dao.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("permissions.*").
		Join("JOIN permissions").
		JoinOn("permissions.kind = ?", "Volume").
		JoinOn("permissions.resource_id = ?TableAlias.id").
		Apply(filter.Filter).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("user has no volumes")
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) CreateVolume(ctx context.Context, vol *model.Volume) error {
	dao.log.Debugf("create volume %+v", vol)

	_, err := dao.db.Model(vol).
		Returning("*").
		Insert()
	return dao.handleError(err)
}
