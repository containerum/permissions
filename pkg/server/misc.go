package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/containerum/bill-external/errors"
	billing "github.com/containerum/bill-external/models"
	"github.com/containerum/utils/httputil"
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

func NamespaceVolumeGlusterLabel(nsLabel string) string {
	return fmt.Sprintf("%s-volume", nsLabel)
}

// VolumeGlusterName generates volume name for glusterfs (non-persistent volumes)
func VolumeGlusterName(nsLabel, userID string) string {
	glusterName := sha256.Sum256([]byte(fmt.Sprintf("%s-volume%s", nsLabel, userID)))
	return hex.EncodeToString(glusterName[:])
}

func OwnerCheck(ctx context.Context, resource model.Resource) error {
	if httputil.MustGetUserID(ctx) != resource.OwnerUserID && !IsAdminRole(ctx) {
		return errors.ErrPermissionDenied().AddDetailF("only resource owner can do this")
	}
	return nil
}
