package model

import (
	"github.com/containerum/kube-client/pkg/model"
)

// Volume describes volume. It`s here for migrations compatibility.
//
// swagger:ignore
type Volume struct {
	tableName struct{} `sql:"volumes"`

	Resource

	Capacity int `sql:"capacity,notnull" json:"capacity"`

	// swagger:strfmt uuid
	NamespaceID string `sql:"ns_id,type:uuid" json:"namespace_id,omitempty"`

	GlusterName string `sql:"gluster_name,notnull" json:"gluster_name,omitempty"`

	// swagger:strfmt uuid
	StorageID string `sql:"storage_id,type:uuid,notnull" json:"storage_id,omitempty"`

	StorageName string `sql:"-" json:"storage_name,omitempty"`

	AccessMode model.PersistentVolumeAccessMode `sql:"access_mode,notnull" json:"access_mode,omitempty"`
}
