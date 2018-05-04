package server

import (
	"context"

	"git.containerum.net/ch/utils/httputil"
	"github.com/containerum/bill-external/errors"
	billing "github.com/containerum/bill-external/models"
)

// IsAdminRole checks that request came from user with admin permissions.
func IsAdminRole(ctx context.Context) bool {
	if v, ok := ctx.Value(httputil.UserRoleContextKey).(string); ok {
		return v == "admin"
	}
	return false
}

// CheckTariff checks if user has permissions to use tariff
func CheckTariff(tariff billing.Tariff, isAdmin bool) error {
	if !tariff.Active {
		return errors.ErrPermissionDenied().AddDetailF("tariff is not active")
	}
	if !isAdmin && !tariff.Public {
		return errors.ErrPermissionDenied().AddDetailF("tariff is not public")
	}

	return nil
}
