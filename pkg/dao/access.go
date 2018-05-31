package dao

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/model"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/go-pg/pg/orm"
)

type AccessWithLabel struct {
	model.Permission `pg:",override"`

	Label string `sql:"label"`
}

func (dao *DAO) UserAccesses(ctx context.Context, userID string) ([]AccessWithLabel, error) {
	dao.log.WithField("user_id", userID).Debugf("get accesses")

	var ret []AccessWithLabel
	err := dao.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		ColumnExpr("ns.label AS label").
		Join("LEFT JOIN namespaces AS ns").JoinOn("?TableAlias.resource_id = ns.id").JoinOn("?TableAlias.resource_type = ?", model.ResourceNamespace).
		Where("?TableAlias.user_id = ?", userID).
		WhereGroup(func(query *orm.Query) (*orm.Query, error) {
			return query.
				Where("ns.label IS NOT NULL"), nil
		}).
		Select()
	if err != nil {
		return nil, dao.handleError(err)
	}

	return ret, nil
}

func (dao *DAO) SetUserAccesses(ctx context.Context, userID string, level kubeClientModel.AccessLevel) error {
	dao.log.WithField("user_id", userID).Debugf("set accesses to %s", level)

	nsIDsQuery := dao.db.Model(&model.Namespace{}).Column("id").Where("owner_user_id = ?", userID)
	volIDsQuery := dao.db.Model(&model.Volume{}).Column("id").Where("owner_user_id = ?", userID)

	// We can lower initial access lever, upper current access level (but not greater then initial) or set to initial
	_, err := dao.db.Model(&model.Permission{}).
		Where("resource_id IN (? UNION ALL ?)", nsIDsQuery, volIDsQuery).
		Set(`current_access_level = CASE WHEN current_access_level > ?0 THEN ?0
											WHEN current_access_level <= ?0 AND initial_access_level > ?0 THEN ?0
											ELSE initial_access_level END`, level).
		Update()
	if err != nil {
		return dao.handleError(err)
	}

	return nil
}

func (dao *DAO) setResourceAccess(ctx context.Context, permission model.Permission) error {
	_, err := dao.db.Model(&permission).
		OnConflict(`(resource_type, resource_id, user_id) DO UPDATE`).
		Set(`initial_access_level = ?initial_access_level`).
		Set(`current_access_level = LEAST(?initial_access_level, ?current_access_level)::ACCESS_LEVEL`).
		Insert()

	if err != nil {
		return dao.handleError(err)
	}

	return nil
}

func (dao *DAO) SetNamespaceAccess(ctx context.Context, ns model.Namespace, accessLevel kubeClientModel.AccessLevel, toUserID string) error {
	dao.log.WithField("ns_id", ns.ID).Debugf("set namespace access %s to %s", accessLevel, toUserID)

	return dao.setResourceAccess(ctx, model.Permission{
		ResourceType:       model.ResourceNamespace,
		ResourceID:         ns.ID,
		UserID:             toUserID,
		InitialAccessLevel: accessLevel,
		CurrentAccessLevel: accessLevel,
	})
}

func (dao *DAO) deleteResourceAccess(ctx context.Context, resource model.Resource, kind model.ResourceType, userID string) error {
	_, err := dao.db.Model(&model.Permission{UserID: userID, ResourceID: resource.ID, ResourceType: kind}).
		Where("user_id = ?user_id").
		Where("resource_type = ?resource_type").
		Where("resource_id = ?resource_id").
		Where("initial_access_level < ?", kubeClientModel.Owner). // do not delete owner permission
		Delete()

	if err != nil {
		return dao.handleError(err)
	}

	return nil
}

func (dao *DAO) DeleteNamespaceAccess(ctx context.Context, ns model.Namespace, userID string) error {
	dao.log.WithField("ns_id", ns.ID).Debugf("delete namespace access to user %s", userID)

	return dao.deleteResourceAccess(ctx, ns.Resource, model.ResourceNamespace, userID)
}
