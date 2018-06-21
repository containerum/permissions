package router

import (
	"net/http"

	"git.containerum.net/ch/permissions/pkg/model"
	"git.containerum.net/ch/permissions/pkg/server"
	"github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type accessHandlers struct {
	tv   *TranslateValidate
	acts server.AccessActions
}

func (ah *accessHandlers) getUserAccessesHandler(ctx *gin.Context) {
	ret, err := ah.acts.GetUserAccesses(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, ret)
}

func (ah *accessHandlers) setUserAccessesHandler(ctx *gin.Context) {
	var req model.SetUserAccessesRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.SetUserAccesses(ctx.Request.Context(), req.Access); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) setNamespaceAccessHandler(ctx *gin.Context) {
	var req model.SetUserAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.SetNamespaceAccess(ctx.Request.Context(), ctx.Param("id"), req.Username, req.Access); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) deleteNamespaceAccessHandler(ctx *gin.Context) {
	var req model.DeleteUserAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.DeleteNamespaceAccess(ctx.Request.Context(), ctx.Param("id"), req.UserName); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) getNamespaceAccessesHandler(ctx *gin.Context) {
	ret, err := ah.acts.GetNamespaceAccesses(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	httputil.MaskForNonAdmin(ctx, &ret)
	ctx.JSON(http.StatusOK, ret)
}

func (r *Router) SetupAccessRoutes(acts server.AccessActions) {
	handlers := &accessHandlers{acts: acts, tv: r.tv}

	// swagger:operation GET /accesses Permissions GetResourcesAccesses
	//
	// Returns user accesses to resources.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	// responses:
	//	 '200':
	//     description: user accesses to resources (TODO: schema)
	//	 default:
	//	   $ref: '#/responses/error'
	r.engine.GET("/accesses", handlers.getUserAccessesHandler)

	// swagger:operation PUT /admin/accesses Permissions SetResourcesAccesses
	//
	// Assign access level for all user resources. Used for billing purposes.
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
	//      $ref: "#/definitions/SetResourcesAccessesRequest"
	// responses:
	//	 '200':
	//	   description: access set
	//	 default:
	//	   $ref: '#/responses/error'
	r.engine.PUT("/admin/accesses", handlers.setUserAccessesHandler)

	// swagger:operation PUT /namespaces/{id}/accesses Permissions SetNamespaceAccess
	//
	// Grant namespace permission to user.
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
	//      $ref: "#/definitions/SetResourceAccessRequest"
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//	 '200':
	//	   description: access set
	//	 default:
	//	   $ref: '#/responses/error'
	r.engine.PUT("/namespaces/:id/accesses", handlers.setNamespaceAccessHandler)

	// swagger:operation DELETE /namespaces/{id}/accesses Permissions DeleteNamespaceAccess
	//
	// Delete namespace permission to user.
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
	//      $ref: "#/definitions/DeleteResourceAccessRequest"
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//	 '200':
	//	   description: access deleted
	//	 default:
	//	   $ref: '#/responses/error'
	r.engine.DELETE("/namespaces/:id/accesses", handlers.deleteNamespaceAccessHandler)

	// swagger:operation GET /namespaces/{id}/accesses Permissions GetNamespaceWithPermissions
	//
	// Get namespace with user permissions.
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
	r.engine.GET("/namespaces/:id/accesses", handlers.getNamespaceAccessesHandler)
}
