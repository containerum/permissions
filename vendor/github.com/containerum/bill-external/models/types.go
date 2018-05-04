package models

import (
	"time"
)

// swagger:ignore
type ResourceType string

const (
	Namespace ResourceType = "namespace"
	Volume    ResourceType = "volume"
)

// Tariff represents generic billing tariff for resource
// If tariff is public it available for all users.
// If tariff not public it available only for admins.
//
// swagger:ignore
type Tariff struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	Price       float64   `json:"price"`
	Active      bool      `json:"is_active"`
	Public      bool      `json:"is_public"`
	BillingID   int       `json:"billing_id"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NamespaceTariff represents billing tariff for namespace
// If VolumeSize provided non-persistent volume will be created.
//
// swagger:model
type NamespaceTariff struct {
	Tariff

	CPULimit         int     `json:"cpu_limit"`
	MemoryLimit      int     `json:"memory_limit"`
	Traffic          int     `json:"traffic"`       // in gigabytes
	TrafficPrice     float64 `json:"traffic_price"` // price of overused traffic.
	ExternalServices int     `json:"external_services"`
	InternalServices int     `json:"internal_services"`
	VolumeSize       int     `json:"volume_size"`
}

// VolumeTariff represents billing tariff for (persistent) volume
//
// swagger:model
type VolumeTariff struct {
	Tariff

	StorageLimit  int `json:"storage_limit"` // in gigabytes
	ReplicasLimit int `json:"replicas_limit"`
}
