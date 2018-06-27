package router

import (
	"net/http"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"git.containerum.net/ch/permissions/pkg/server"
	"github.com/containerum/cherry/adaptors/gonic"
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

	if err := nh.acts.CreateNamespace(ctx.Request.Context(), ctx.Param("project"), req); err != nil {
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

	if err := nh.acts.AdminResizeNamespace(ctx.Request.Context(), ctx.Param("namespace"), req); err != nil {
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

	if err := nh.acts.RenameNamespace(ctx.Request.Context(), ctx.Param("project"), ctx.Param("namespace"), req.Label); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (nh *namespaceHandlers) deleteNamespaceHandler(ctx *gin.Context) {
	if err := nh.acts.DeleteNamespace(ctx.Request.Context(), ctx.Param("project"), ctx.Param("namespace")); err != nil {
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
	ret, err := nh.acts.GetNamespace(ctx.Request.Context(), ctx.Param("project"), ctx.Param("namespace"))
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

	if err := nh.acts.ResizeNamespace(ctx.Request.Context(), ctx.Param("project"), ctx.Param("namespace"), req.TariffID); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
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

	// swagger:operation POST /projects/:project/namespaces Namespaces CreateNamespace
	//
	// Create namespace using billing.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ProjectID'
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
	r.engine.POST("/projects/:project/namespaces", handlers.createNamespaceHandler)

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
	//  - $ref: '#/parameters/NamespaceID'
	// responses:
	//   '200':
	//     description: namespace resized
	//   default:
	//     $ref: '#/responses/error'
	r.engine.PUT("/admin/namespaces/:namespace", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.adminResizeNamespaceHandler)

	// swagger:operation PUT /projects/{project}/namespaces/{namespace}/rename Namespaces RenameNamespace
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
	//  - $ref: '#/parameters/NamespaceID'
	//  - $ref: '#/parameters/ProjectID'
	// responses:
	//   '200':
	//     description: namespace renamed
	//   default:
	//     $ref: '#/responses/error'
	r.engine.PUT("/projects/:project/namespaces/:namespace/rename", handlers.renameNamespaceHandler)

	// swagger:operation PUT /projects/{project}/namespaces/{namespace} Namespaces ResizeNamespace
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
	//  - $ref: '#/parameters/NamespaceID'
	//  - $ref: '#/parameters/ProjectID'
	// responses:
	//   '200':
	//     description: namespace resized
	//   default:
	//     $ref: '#/responses/error'
	r.engine.PUT("/projects/:project/namespaces/:namespace", handlers.resizeNamespaceHandler)

	// swagger:operation DELETE /projects/{project}/namespaces/{namespace} Namespaces DeleteNamespace
	//
	// Delete namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	//  - $ref: '#/parameters/ProjectID'
	// responses:
	//   '200':
	//     description: namespace deleted
	//   default:
	//     $ref: '#/responses/error'
	r.engine.DELETE("/projects/:project/namespaces/:namespace", handlers.deleteNamespaceHandler)

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

	// swagger:operation GET /projects/{project}/namespaces/{namespace} Namespaces GetNamespace
	//
	// Get namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	//  - $ref: '#/parameters/ProjectID'
	// responses:
	//   '200':
	//     description: namespace response
	//     schema:
	//       $ref: '#/definitions/Namespace'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/projects/:project/namespaces/:namespace", handlers.getNamespaceHandler)

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
}
