package model

// Storage describes volumes storage. It`s here for migrations compatibility.
//
// swagger:ignore
type Storage struct {
	tableName struct{} `sql:"storages"`

	// swagger:strfmt uuid
	ID string `sql:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id,omitempty"`

	Name string `sql:"name,notnull,unique" json:"name"`

	Size int `sql:"size,notnull" json:"size"`

	Used int `sql:"used,notnull" json:"used"`

	Replicas int `sql:"replicas,notnull" json:"replicas"`

	IPs []string `sql:"ips,notnull,type:inet[],array" json:"ips"`

	Volumes []*Volume `pg:"fk:storage_id" sql:"-" json:"volumes"`
}
