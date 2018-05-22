package dao

import (
	"context"
	"reflect"
	"time"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
)

type NamespaceFilter struct {
	orm.Pager
	NotDeleted bool `filter:"not_deleted"`
	Deleted    bool `filter:"deleted"`
	NotLimited bool `filter:"not_limited"`
	Limited    bool `filter:"limited"`
	Owned      bool `filter:"owner"`
	NotOwned   bool `filter:"not_owner"`
}

var nsFilterCache = make(map[string]int)

func init() {
	t := reflect.TypeOf(NamespaceFilter{})
	for i := 0; i < t.NumField(); i++ {
		tag, ok := t.Field(i).Tag.Lookup("filter")
		if !ok {
			continue
		}
		nsFilterCache[tag] = i
	}
}

func ParseNamespaceFilter(filters ...string) NamespaceFilter {
	var ret NamespaceFilter
	v := reflect.ValueOf(&ret).Elem()
	for _, filter := range filters {
		if field, ok := nsFilterCache[filter]; ok {
			v.Field(field).SetBool(true)
		}
	}
	return ret
}

func (f *NamespaceFilter) Filter(q *orm.Query) (*orm.Query, error) {
	if f.NotDeleted {
		q = q.Where("NOT ?TableAlias.deleted")
	}
	if f.Deleted {
		q = q.Where("?TableAlias.deleted")
	}
	if f.NotLimited {
		q = q.Where("permission.initial_access_level = permissions.current_access_level")
	}
	if f.Limited {
		q = q.Where("permission.initial_access_level != permissions.initial_access_level")
	}
	if f.Owned {
		q = q.Where("permission.initial_access_level = ?", model.AccessOwner)
	}
	if f.NotOwned {
		q = q.Where("permission.initial_access_level != ?", model.AccessOwner)
	}

	return q.Apply(f.Paginate), nil
}

func (dao *DAO) NamespaceByID(ctx context.Context, userID, id string) (ret model.NamespaceWithPermissions, err error) {
	dao.log.WithFields(logrus.Fields{
		"id":      id,
		"user_id": userID,
	}).Debugf("get namespace by id")

	ret.ID = id
	err = dao.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("Permission").
		WherePK().
		Where("permission.resource_id = ?TableAlias.id").
		Where("permission.user_id = ?", userID).
		Where("coalesce(permission.current_access_level, ?0) > ?0", model.AccessNone).
		Where("NOT ?TableAlias.deleted").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("namespace with id %s not exists", id)
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) NamespaceByLabel(ctx context.Context, userID, label string) (ret model.NamespaceWithPermissions, err error) {
	dao.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("get namespace by user id and label")

	err = dao.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("Permission").
		Where("permission.user_id = ?", userID).
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

func (dao *DAO) NamespaceVolumes(ctx context.Context, ns *model.Namespace) (err error) {
	dao.log.WithFields(logrus.Fields{
		"owner_user_id": ns.OwnerUserID,
		"label":         ns.Label,
	}).Debugf("get namespace volumes")

	err = dao.db.Model(ns).
		Column("Volumes").
		WherePK().
		Where("NOT deleted").
		Relation("Volumes").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = nil // no volumes is ok
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) NamespacePermissions(ctx context.Context, ns *model.NamespaceWithPermissions) error {
	dao.log.WithFields(logrus.Fields{
		"owner_user_id": ns.OwnerUserID,
		"label":         ns.Label,
	}).Debugf("get namespace permissions")

	err := dao.db.Model(ns).
		WherePK().
		Column("Permissions").
		Relation("Permissions", func(q *orm.Query) (*orm.Query, error) {
			return q.Where("initial_access_level != ?", model.AccessOwner), nil
		}).
		Select()
	if len(ns.Permissions) == 0 {
		ns.Permissions = make([]model.Permission, 0)
	}
	switch err {
	case pg.ErrNoRows:
		return nil
	default:
		return dao.handleError(err)
	}
}

func (dao *DAO) UserNamespaces(ctx context.Context, userID string, filter NamespaceFilter) (ret []model.NamespaceWithPermissions, err error) {
	dao.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filter,
	}).Debugf("get user namespaces")

	ret = make([]model.NamespaceWithPermissions, 0)

	err = dao.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("Volumes", "Permission").
		Where("permission.user_id = ?", userID).
		Apply(filter.Filter).
		Relation("Volumes").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("user has no namespaces")
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) AllNamespaces(ctx context.Context, filter NamespaceFilter) (ret []model.NamespaceWithPermissions, err error) {
	dao.log.Debugf("get all namespaces")

	ret = make([]model.NamespaceWithPermissions, 0)

	err = dao.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("Volumes", "Permission").
		Relation("Volumes").
		Apply(filter.Filter).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("no namespaces in system")
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) CreateNamespace(ctx context.Context, namespace *model.Namespace) error {
	dao.log.Debugf("create namespace %+v", namespace)

	_, err := dao.db.Model(namespace).
		Returning("*").
		Insert()
	if err != nil {
		err = dao.handleError(err)
	}

	return err
}

func (dao *DAO) RenameNamespace(ctx context.Context, namespace *model.Namespace, newLabel string) error {
	dao.log.WithField("new_label", newLabel).Debugf("rename namespace %+v", namespace)

	cnt, err := dao.db.Model(namespace).
		Where("owner_user_id = ?owner_user_id").
		Where("label = ?", newLabel).
		Where("NOT deleted").
		Count()
	if err != nil {
		return dao.handleError(err)
	}
	if cnt > 0 {
		return errors.ErrResourceAlreadyExists().AddDetailF("namespace %s already exists", newLabel)
	}

	_, err = dao.db.Model(namespace).
		WherePK().
		Set("label = ?", newLabel).
		Returning("*").
		Update()
	return dao.handleError(err)
}

func (dao *DAO) ResizeNamespace(ctx context.Context, namespace model.Namespace) error {
	dao.log.Debugf("resize namespace %+v", namespace)

	result, err := dao.db.Model(&namespace).
		WherePK().
		Set("cpu = ?cpu").
		Set("ram = ?ram").
		Set("max_ext_services = ?max_ext_services").
		Set("max_int_services = ?max_int_services").
		Set("max_traffic = ?max_traffic").
		Set("tariff_id = ?tariff_id").
		Update()
	if err != nil {
		return dao.handleError(err)
	}

	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("namespace %s not exists for user", namespace.Label)
	}

	return nil
}

func (dao *DAO) DeleteNamespace(ctx context.Context, namespace *model.Namespace) error {
	dao.log.Debugf("delete namespace %+v", namespace)

	namespace.Deleted = true
	now := time.Now().UTC()
	namespace.DeleteTime = &now

	result, err := dao.db.Model(namespace).
		Where("NOT deleted").
		WherePK().
		Set("deleted = ?deleted").
		Set("delete_time = ?delete_time").
		Returning("*").
		Update()
	if err != nil {
		return dao.handleError(err)
	}

	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("namespace %s not exists", namespace.Label)
	}

	return nil
}

func (dao *DAO) DeleteNamespaceVolumes(ctx context.Context, namespace model.Namespace) (deleted []model.Volume, err error) {
	dao.log.Debugf("delete namespace volumes %+v", namespace)

	deleted = make([]model.Volume, 0)

	_, err = dao.db.Model(&deleted).
		Where("ns_id = ?", namespace.ID).
		Where("NOT deleted").
		Set("active = FALSE").
		Set("deleted = TRUE").
		Set("delete_time = now()").
		Returning("*").
		Update()
	err = dao.handleError(err)
	return
}

func (dao *DAO) DeleteAllUserNamespaces(ctx context.Context, userID string) (deleted []model.Namespace, err error) {
	dao.log.WithField("user_id", userID).Debugf("delete user namespaces")

	deleted = make([]model.Namespace, 0)

	result, err := dao.db.Model(&deleted).
		Where("owner_user_id = ?", userID).
		Where("NOT deleted").
		Set("deleted = TRUE").
		Set("delete_time = now()").
		Returning("*").
		Update()
	if err != nil {
		err = dao.handleError(err)
		return
	}
	if result.RowsAffected() <= 0 {
		err = errors.ErrResourceNotExists().AddDetailF("user %s has no namespaces", userID)
		return
	}

	return
}

func (dao *DAO) DeleteAllUserNamespaceVolumes(ctx context.Context, userID string) (deleted []model.Volume, err error) {
	dao.log.WithField("user_id", userID).Debugf("delete user namespace volumes")

	deleted = make([]model.Volume, 0)

	_, err = dao.db.Model(&deleted).
		Where("owner_user_id = ?", userID).
		Where("NOT deleted").
		Set("active = FALSE").
		Set("deleted = TRUE").
		Set("delete_time = now()").
		Returning("*").
		Update()
	err = dao.handleError(err)
	return
}
