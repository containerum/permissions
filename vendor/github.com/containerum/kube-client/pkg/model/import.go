package model

const ImportSuccessfulMessage = "Successfully imported"

// ImportResponseTotal -- import response for all services
//
// swagger:model
type ImportResponseTotal map[string]ImportResponse

// ImportResponse -- response after resources import
//
// swagger:model
type ImportResponse struct {
	Imported []ImportResult `json:"imported" yaml:"imported"`
	Failed   []ImportResult `json:"failed" yaml:"failed"`
}

// ImportResult -- import result for one resource
//
// swagger:model
type ImportResult struct {
	Name    string `json:"name" yaml:"name"`
	Message string `json:"message" yaml:"message"`
}

func (resp *ImportResponse) ImportSuccessful(name string) {
	resp.Imported = append(resp.Imported, ImportResult{
		Name:    name,
		Message: ImportSuccessfulMessage,
	})
}

func (resp *ImportResponse) ImportFailed(name, message string) {
	resp.Failed = append(resp.Failed, ImportResult{
		Name:    name,
		Message: message,
	})
}
