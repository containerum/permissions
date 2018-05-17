package models

// SubscribeTariffRequest contains parameters needed for tariff subscribing
//
// swagger:model
type SubscribeTariffRequest struct {
	TariffID      string       `json:"tariff_id"`
	ResourceType  ResourceType `json:"resource_type"`
	ResourceLabel string       `json:"resource_label"`
	ResourceID    string       `json:"resource_id"`
}

// ChangeTariffRequest contains parameters needed for tariff changing
//
// swagger:model
type ChangeTariffRequest struct {
	TariffID      string       `json:"tariff_id"`
}

// MassiveUnsubscribeTariffRequest contains parameters needed for all tariffs unsubscribing
//
// swagger:model
type MassiveUnsubscribeTariffRequest struct {
	Resources []string `json:"resources"`
}

// RenameRequest contains parameters needed for resource renaming
//
// swagger:model
type RenameRequest struct {
	ResourceLabel      string       `json:"resource_label"`
}