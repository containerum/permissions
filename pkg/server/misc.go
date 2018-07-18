package server

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/clients"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/containerum/bill-external/errors"
	billing "github.com/containerum/bill-external/models"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
)

const DefaultVolumeName = "default-volume"

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

func OwnerCheck(ctx context.Context, resource model.Resource) error {
	if httputil.MustGetUserID(ctx) != resource.OwnerUserID && !IsAdminRole(ctx) {
		return errors.ErrPermissionDenied().AddDetailF("only resource owner can do this")
	}
	return nil
}

func AddOwnerLogin(ctx context.Context, r *model.Resource, client clients.UserManagerClient) error {
	user, err := client.UserInfoByID(ctx, r.OwnerUserID)
	if err != nil {
		return err
	}
	r.OwnerUserLogin = user.Login
	return nil
}

func AddUserLogins(ctx context.Context, permissions []model.Permission, client clients.UserManagerClient) error {
	if len(permissions) == 0 {
		return nil
	}
	userIDs := make([]string, len(permissions))
	for i := range permissions {
		userIDs[i] = permissions[i].UserID
	}
	userLogins, err := client.UserLoginIDList(ctx, userIDs...)
	if err != nil {
		return err
	}

	for i := range permissions {
		permissions[i].UserLogin = userLogins[permissions[i].UserID]
	}
	return nil
}

func NamespaceAddUsage(ctx context.Context, ns *kubeClientModel.Namespace, client clients.KubeAPIClient) error {
	kubeNS, err := client.GetNamespace(ctx, ns.ID)
	if err != nil {
		return err
	}
	ns.Resources.Used = kubeNS.Resources.Used
	return nil
}

func UserGroupAccessToDBAccess(access kubeClientModel.UserGroupAccess) kubeClientModel.AccessLevel {
	switch access {
	case kubeClientModel.OwnerAccess:
		return kubeClientModel.Owner
	case kubeClientModel.MasterAccess, kubeClientModel.AdminAccess:
		return kubeClientModel.Write
	case kubeClientModel.MemberAccess:
		return kubeClientModel.ReadDelete
	case kubeClientModel.GuestAccess:
		return kubeClientModel.Read
	default:
		return kubeClientModel.None
	}
}
