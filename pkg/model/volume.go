package model

import "github.com/go-pg/pg/orm"

// swagger:ignore
type Volume struct {
	tableName struct{} `sql:"volumes"`

	Resource

	Active      bool   `sql:"active,notnull"`
	Capacity    int    `sql:"capacity,notnull"`
	Replicas    int    `sql:"replicas,notnull"`
	NamespaceID int    `sql:"ns_id,type:UUID,notnull"`
	GlusterName string `sql:"gluster_name,notnull"`
	StorageID   string `sql:"storage_id,type:UUID,notnull"`

	Permission []*Permission `pg:"polymorphic:resource_" sql:"-"`
}

func (v *Volume) BeforeUpdate(db orm.DB) error {
	if err := v.Resource.BeforeUpdate(db); err != nil {
		return err
	}

	var err error
	if v.Deleted {
		_, err = db.Model(&Storage{}).
			Relation("Volumes").
			Set("used = used - ?", v.Capacity).
			Update()
	} else {
		oldCapacityQuery := db.Model(v).Column("capacity").Where("id = ?", v.ID)
		_, err = db.Model(&Storage{}).
			Set("used = used - (?) + ?", oldCapacityQuery, v.Capacity).
			Update(v)
	}
	return err
}

func (v *Volume) AfterInsert(db orm.DB) error {
	return db.Insert(&Permission{
		ResourceID:         v.ID,
		UserID:             v.OwnerUserID,
		ResourceKind:       "Volume",
		InitialAccessLevel: AccessOwner,
		CurrentAccessLevel: AccessOwner,
	})
}
