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

// UnsubscribeTariffRequest contains parameters needed for tariff unsubscribing
//
// swagger:model
type UnsubscribeTariffRequest struct {
	ResourceID string `json:"resource_id"`
}
