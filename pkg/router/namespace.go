package router

import (
	"net/http"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"git.containerum.net/ch/permissions/pkg/server"
	"github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type namespaceHandlers struct {
	tv   *TranslateValidate
	acts server.NamespaceActions
}

func (nh *namespaceHandlers) adminCreateNamespace(ctx *gin.Context) {
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

func (nh *namespaceHandlers) adminResizeNamespace(ctx *gin.Context) {
	var req model.NamespaceAdminResizeRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.BadRequest(ctx, err))
		return
	}

	if err := nh.acts.AdminResizeNamespace(ctx.Request.Context(), ctx.Param("label"), req); err != nil {
		ctx.AbortWithStatusJSON(nh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (nh *namespaceHandlers) deleteNamespaceHandler(ctx *gin.Context) {
	if err := nh.acts.DeleteNamespace(ctx.Request.Context(), ctx.Param("label")); err != nil {
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

func (r *Router) SetupNamespaceRoutes(acts server.NamespaceActions) {
	handlers := &namespaceHandlers{tv: r.tv, acts: acts}

	// swagger:operation POST /admin/namespaces Namespaces AdminCreateNamespace
	//
	// Create namespace without billing.
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
	r.engine.POST("/admin/namespaces", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.adminCreateNamespace)

	// swagger:operation PUT /admin/namespaces/{label} Namespaces AdminResizeNamespace
	//
	// Resize namespace without billing.
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
	//  - name: label
	//    in: path
	//    required: true
	//    type: string
	// responses:
	//   '200':
	//     description: namespace resized
	//   default:
	//     $ref: '#/responses/error'
	r.engine.PUT("/admin/namespaces/:label", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.adminResizeNamespace)

	// swagger:operation DELETE /namespaces/{label} Namespaces DeleteNamespace
	//
	// Delete namespace.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: label
	//    in: path
	//    required: true
	//    type: string
	// responses:
	//   '200':
	//     description: namespace deleted
	//   default:
	//     $ref: '#/responses/error'
	r.engine.DELETE("/namespaces/:label", handlers.deleteNamespaceHandler)

	// swagger:operation DELETE /admin/namespaces Namespaces DeleteAllUserNamespaces
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
	r.engine.DELETE("/admin/namespaces", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.deleteAllUserNamespacesHandler)
}
