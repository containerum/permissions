package router

import (
	"net/http"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"git.containerum.net/ch/permissions/pkg/server"
	"github.com/containerum/cherry/adaptors/gonic"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type namespaceHandlers struct {
	tv   *TranslateValidate
	acts server.NamespaceActions
}

func (nh *namespaceHandlers) adminCreateNamespaceHandler(ctx *gin.Context) {
	var req model.NamespaceAdminCreateRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	if err := nh.acts.AdminCreateNamespace(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (nh *namespaceHandlers) createNamespaceHandler(ctx *gin.Context) {
	var req model.NamespaceCreateRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	if err := nh.acts.CreateNamespace(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (nh *namespaceHandlers) adminResizeNamespaceHandler(ctx *gin.Context) {
	var req model.NamespaceAdminResizeRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	if err := nh.acts.AdminResizeNamespace(ctx.Request.Context(), ctx.Param("id"), req); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (nh *namespaceHandlers) renameNamespaceHandler(ctx *gin.Context) {
	var req model.NamespaceRenameRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	if err := nh.acts.RenameNamespace(ctx.Request.Context(), ctx.Param("id"), req.Label); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (nh *namespaceHandlers) deleteNamespaceHandler(ctx *gin.Context) {
	if err := nh.acts.DeleteNamespace(ctx.Request.Context(), ctx.Param("id")); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (nh *namespaceHandlers) deleteAllUserNamespacesHandler(ctx *gin.Context) {
	if err := nh.acts.DeleteAllUserNamespaces(ctx.Request.Context()); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (nh *namespaceHandlers) getNamespaceHandler(ctx *gin.Context) {
	ret, err := nh.acts.GetNamespace(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}
	httputil.MaskForNonAdmin(ctx, &ret)
	ctx.JSON(http.StatusOK, ret)
}

func (nh *namespaceHandlers) getUserNamespacesHandler(ctx *gin.Context) {
	ret, err := nh.acts.GetUserNamespaces(ctx.Request.Context(), getFilters(ctx.Request.URL.Query())...)
	if err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}
	for i := range ret {
		httputil.MaskForNonAdmin(ctx, &ret[i])
	}
	ctx.JSON(http.StatusOK, gin.H{"namespaces": ret})
}

func (nh *namespaceHandlers) getAllNamespacesHandler(ctx *gin.Context) {
	page, perPage, err := getPaginationParams(ctx.Request.URL.Query())
	if err != nil {
		gonic.Gonic(errors.ErrRequestValidationFailed().AddDetailsErr(err), ctx)
		return
	}
	ret, err := nh.acts.GetAllNamespaces(ctx.Request.Context(), page, perPage, getFilters(ctx.Request.URL.Query())...)
	if err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"namespaces": ret})
}

func (nh *namespaceHandlers) resizeNamespaceHandler(ctx *gin.Context) {
	var req model.NamespaceResizeRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	if err := nh.acts.ResizeNamespace(ctx.Request.Context(), ctx.Param("id"), req.TariffID); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (nh *namespaceHandlers) addGroupToNamespaceHandler(ctx *gin.Context) {
	var req model.ProjectAddGroupRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	if err := nh.acts.AddGroupNamespace(ctx.Request.Context(), ctx.Param("id"), req.GroupID); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

func (nh *namespaceHandlers) setGroupMemberNamespaceAccessHandler(ctx *gin.Context) {
	var req model.SetGroupMemberAccessRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	if err := nh.acts.SetGroupMemberNamespaceAccess(ctx.Request.Context(), ctx.Param("id"), ctx.Param("group"), req); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

func (nh *namespaceHandlers) getNamespaceGroupsHandler(ctx *gin.Context) {
	groups, err := nh.acts.GetNamespaceGroups(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (nh *namespaceHandlers) deleteGroupFromNamespaceHandler(ctx *gin.Context) {
	if err := nh.acts.DeleteGroupFromNamespace(ctx.Request.Context(), ctx.Param("id"), ctx.Param("group")); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

func (nh *namespaceHandlers) getGroupNamespacesHandler(ctx *gin.Context) {
	ret, err := nh.acts.GetGroupsNamespaces(ctx.Request.Context(), ctx.Param("group"))
	if err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}
	for i := range ret {
		httputil.MaskForNonAdmin(ctx, &ret[i])
	}
	ctx.JSON(http.StatusOK, gin.H{"namespaces": ret})
}

func (nh *namespaceHandlers) importNamespacesHandler(ctx *gin.Context) {
	var req kubeClientModel.NamespacesList

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	ctx.JSON(http.StatusAccepted, nh.acts.ImportNamespaces(ctx.Request.Context(), req))
}

func (r *Router) SetupNamespaceRoutes(acts server.NamespaceActions) {
	handlers := &namespaceHandlers{tv: r.tv, acts: acts}

	// swagger:operation POST /admin/namespaces Namespaces AdminCreateNamespace
	//
	// Create namespace without billing (admin only).
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/NamespaceAdminCreateRequest'
	// responses:
	//   '201':
	//     description: namespace created
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/admin/namespaces", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.adminCreateNamespaceHandler)

	// swagger:operation POST /namespaces Namespaces CreateNamespace
	//
	// Create namespace using billing.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/NamespaceCreateRequest'
	// responses:
	//   '201':
	//     description: namespace created
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/namespaces", handlers.createNamespaceHandler)

	// swagger:operation POST /import/namespaces Namespaces ImportNamespaces
	//
	// Import namespaces without creating permissions or in kube-api.
	//
	// ---
	// parameters:
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/NamespacesList'
	// responses:
	//   '201':
	//     description: namespace created
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/import/namespaces", handlers.importNamespacesHandler)

	// swagger:operation PUT /admin/namespaces/{id} Namespaces AdminResizeNamespace
	//
	// Resize namespace without billing (admin only).
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/NamespaceAdminResizeRequest'
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//   '200':
	//     description: namespace resized
	//   default:
	//     $ref: '#/responses/error'
	r.engine.PUT("/admin/namespaces/:id", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.adminResizeNamespaceHandler)

	// swagger:operation PUT /namespaces/{id}/rename Namespaces RenameNamespace
	//
	// Rename namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/NamespaceRenameRequest'
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//   '200':
	//     description: namespace renamed
	//   default:
	//     $ref: '#/responses/error'
	r.engine.PUT("/namespaces/:id/rename", handlers.renameNamespaceHandler)

	// swagger:operation PUT /namespaces/{id} Namespaces ResizeNamespace
	//
	// Resize namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/NamespaceResizeRequest'
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//   '200':
	//     description: namespace resized
	//   default:
	//     $ref: '#/responses/error'
	r.engine.PUT("/namespaces/:id", handlers.resizeNamespaceHandler)

	// swagger:operation DELETE /namespaces/{id} Namespaces DeleteNamespace
	//
	// Delete namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//   '200':
	//     description: namespace deleted
	//   default:
	//     $ref: '#/responses/error'
	r.engine.DELETE("/namespaces/:id", handlers.deleteNamespaceHandler)

	// swagger:operation DELETE /namespaces Namespaces DeleteAllUserNamespaces
	//
	// Delete all user namespaces.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	// responses:
	//   '200':
	//     description: namespaces deleted
	//   default:
	//     $ref: '#/responses/error'
	r.engine.DELETE("/namespaces", handlers.deleteAllUserNamespacesHandler)

	// swagger:operation GET /namespaces/{id} Namespaces GetNamespace
	//
	// Get namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//   '200':
	//     description: namespace response
	//     schema:
	//       $ref: '#/definitions/Namespace'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/namespaces/:id", handlers.getNamespaceHandler)

	// swagger:operation GET /namespaces Namespaces GetUserNamespaces
	//
	// Get user namespaces.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/Filters'
	// responses:
	//   '200':
	//     description: namespaces response
	//     schema:
	//       type: object
	//       properties:
	//         namespaces:
	//           type: array
	//           items:
	//             $ref: '#/definitions/Namespace'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/namespaces", handlers.getUserNamespacesHandler)

	// swagger:operation GET /admin/namespaces Namespaces GetAllNamespaces
	//
	// Get all namespaces (admin only).
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/Filters'
	//  - $ref: '#/parameters/PageNum'
	//  - $ref: '#/parameters/PerPageLimit'
	// responses:
	//   '200':
	//     description: namespaces response
	//     schema:
	//       type: object
	//       properties:
	//         namespaces:
	//           type: array
	//           items:
	//             $ref: '#/definitions/Namespace'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/admin/namespaces", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.getAllNamespacesHandler)

	// swagger:operation POST /namespaces/{id}/groups Namespaces AddGroupToNamespace
	//
	// Add group to namespace (admin only).
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ResourceID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/ProjectAddGroupRequest'
	// responses:
	//   '202':
	//     description: group added to namespace
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/namespaces/:id/groups", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.addGroupToNamespaceHandler)

	// swagger:operation PUT /namespaces/{id}/groups/{group} Namespaces SetGroupMemberNamespaceAccess
	//
	// Change access of group member to namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ResourceID'
	//  - $ref: '#/parameters/GroupID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/SetGroupMemberAccessRequest'
	// responses:
	//   '202':
	//     description: access set
	//   default:
	//     $ref: '#/responses/error'
	r.engine.PUT("/namespaces/:id/groups/:group", handlers.setGroupMemberNamespaceAccessHandler)

	// swagger:operation GET /namespaces/{id}/groups Namespaces GetNamespaceGroups
	//
	// Get namespace groups.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//   '200':
	//     description: namespace groups
	//     schema:
	//       type: object
	//       properties:
	//         groups:
	//           type: array
	//           items:
	//             $ref: '#/definitions/UserGroup'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/namespaces/:id/groups", handlers.getNamespaceGroupsHandler)

	// swagger:operation DELETE /namespaces/{id}/groups/{group} Namespaces DeleteGroupFromNamespace
	//
	// Delete group permissions from namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ProjectID'
	//  - $ref: '#/parameters/GroupID'
	// responses:
	//   '202':
	//     description: group deleted
	//   default:
	//     $ref: '#/responses/error'
	r.engine.DELETE("/namespaces/:id/groups/:group", handlers.deleteGroupFromNamespaceHandler)

	// swagger:operation GET /groups/{group}/namespaces Namespaces GetGroupNamespaces
	//
	// Get groups namespaces.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/GroupID'
	// responses:
	//   '200':
	//     description: namespaces response
	//     schema:
	//       type: object
	//       properties:
	//         namespaces:
	//           type: array
	//           items:
	//             $ref: '#/definitions/Namespace'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/groups/:group/namespaces", handlers.getGroupNamespacesHandler)
}
