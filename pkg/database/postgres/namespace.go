package postgres

import (
	"context"
	"time"

	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
)

var _ = NamespaceFilter(database.NamespaceFilter{})

func (pgdb *PgDB) NamespaceByID(ctx context.Context, userID, id string) (ret model.NamespaceWithPermissions, err error) {
	pgdb.log.WithFields(logrus.Fields{
		"id":      id,
		"user_id": userID,
	}).Debugf("get namespace by id")

	ret.ID = id
	err = pgdb.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("Permission").
		WherePK().
		Where("permission.resource_id = ?TableAlias.id").
		Where("permission.user_id = ?", userID).
		Where("coalesce(permission.current_access_level, ?0) > ?0", kubeClientModel.None).
		Where("NOT ?TableAlias.deleted").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("namespace with id %s not exists", id)
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) NamespaceByLabel(ctx context.Context, userID, label string) (ret model.NamespaceWithPermissions, err error) {
	pgdb.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("get namespace by user id and label")

	err = pgdb.db.Model(&ret).
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
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) NamespacePermissions(ctx context.Context, ns *model.NamespaceWithPermissions) error {
	pgdb.log.WithFields(logrus.Fields{
		"owner_user_id": ns.OwnerUserID,
		"label":         ns.Label,
	}).Debugf("get namespace permissions")

	err := pgdb.db.Model(ns).
		WherePK().
		Column("Permissions").
		Relation("Permissions", func(q *orm.Query) (*orm.Query, error) {
			return q.Where("initial_access_level != ?", kubeClientModel.Owner), nil
		}).
		Select()
	if len(ns.Permissions) == 0 {
		ns.Permissions = make([]model.Permission, 0)
	}
	switch err {
	case pg.ErrNoRows:
		return nil
	default:
		return pgdb.handleError(err)
	}
}

func (pgdb *PgDB) UserNamespaces(ctx context.Context, userID string, filter database.NamespaceFilter) (ret []model.NamespaceWithPermissions, err error) {
	pgdb.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filter,
	}).Debugf("get user namespaces")

	ret = make([]model.NamespaceWithPermissions, 0)

	f := NamespaceFilter(filter)
	err = pgdb.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Column("Permission").
		Where("permission.user_id = ?", userID).
		Apply(f.Filter).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("user has no namespaces")
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) AllNamespaces(ctx context.Context, filter database.NamespaceFilter) (ret []model.Namespace, err error) {
	pgdb.log.Debugf("get all namespaces")

	ret = make([]model.Namespace, 0)

	f := NamespaceFilter(filter)
	err = pgdb.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Apply(f.Filter).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("no namespaces in system")
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) CreateNamespace(ctx context.Context, namespace *model.Namespace) error {
	pgdb.log.Debugf("create namespace %+v", namespace)

	_, err := pgdb.db.Model(namespace).
		Returning("*").
		Insert()
	if err != nil {
		err = pgdb.handleError(err)
	}

	return err
}

func (pgdb *PgDB) RenameNamespace(ctx context.Context, namespace *model.Namespace, newLabel string) error {
	pgdb.log.WithField("new_label", newLabel).Debugf("rename namespace %+v", namespace)

	cnt, err := pgdb.db.Model(namespace).
		Where("owner_user_id = ?owner_user_id").
		Where("label = ?", newLabel).
		Where("NOT deleted").
		Count()
	if err != nil {
		return pgdb.handleError(err)
	}
	if cnt > 0 {
		return errors.ErrResourceAlreadyExists().AddDetailF("namespace %s already exists", newLabel)
	}

	_, err = pgdb.db.Model(namespace).
		WherePK().
		Set("label = ?", newLabel).
		Returning("*").
		Update()
	return pgdb.handleError(err)
}

func (pgdb *PgDB) ResizeNamespace(ctx context.Context, namespace model.Namespace) error {
	pgdb.log.Debugf("resize namespace %+v", namespace)

	result, err := pgdb.db.Model(&namespace).
		WherePK().
		Set("cpu = ?cpu").
		Set("ram = ?ram").
		Set("max_ext_services = ?max_ext_services").
		Set("max_int_services = ?max_int_services").
		Set("max_traffic = ?max_traffic").
		Set("tariff_id = ?tariff_id").
		Update()
	if err != nil {
		return pgdb.handleError(err)
	}

	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("namespace %s not exists for user", namespace.Label)
	}

	return nil
}

func (pgdb *PgDB) DeleteNamespace(ctx context.Context, namespace *model.Namespace) error {
	pgdb.log.Debugf("delete namespace %+v", namespace)

	namespace.Deleted = true
	now := time.Now().UTC()
	namespace.DeleteTime = &now

	result, err := pgdb.db.Model(namespace).
		Where("NOT deleted").
		WherePK().
		Set("deleted = ?deleted").
		Set("delete_time = ?delete_time").
		Returning("*").
		Update()
	if err != nil {
		return pgdb.handleError(err)
	}

	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("namespace %s not exists", namespace.Label)
	}

	return nil
}

func (pgdb *PgDB) DeleteAllUserNamespaces(ctx context.Context, userID string) (deleted []model.Namespace, err error) {
	pgdb.log.WithField("user_id", userID).Debugf("delete user namespaces")

	deleted = make([]model.Namespace, 0)

	result, err := pgdb.db.Model(&deleted).
		Where("owner_user_id = ?", userID).
		Where("NOT deleted").
		Set("deleted = TRUE").
		Set("delete_time = now()").
		Returning("*").
		Update()
	if err != nil {
		err = pgdb.handleError(err)
		return
	}
	if result.RowsAffected() <= 0 {
		err = errors.ErrResourceNotExists().AddDetailF("user %s has no namespaces", userID)
		return
	}

	return
}

func (pgdb *PgDB) DeleteGroupFromNamespace(ctx context.Context, namespace, groupID string) (deletedPerms []model.Permission, err error) {
	pgdb.log.WithFields(logrus.Fields{
		"namespace": namespace,
		"group_id":  groupID,
	}).Debugf("delete group from namespace")

	_, err = pgdb.db.Model(&deletedPerms).
		Where("group_id = ?", groupID).
		Where("resource_type = ?", model.ResourceNamespace).
		Returning("*").
		Delete()

	switch err {
	case pg.ErrNoRows:
		err = nil
	default:
		err = pgdb.handleError(err)
	}

	return
}
