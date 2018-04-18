package model

import "time"

type Resource struct {
	ID          string     `sql:"id,pk,type:uuid,default:uuid_generate_v4()"`
	CreateTime  time.Time  `sql:"create_time,default:now(),type:TIMESTAMPTZ,notnull"`
	Deleted     bool       `sql:"deleted,notnull"`
	DeleteTime  *time.Time `sql:"delete_time,type:TIMESTAMPTZ"`
	TariffID    string     `sql:"tariff_id,type:uuid,notnull"`
	OwnerUserID string     `sql:"owner_user_id,type:uuid,notnull"`
	Label       string     `sql:"label,notnull"`
}
