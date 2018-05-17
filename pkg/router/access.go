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

	if err := ah.acts.SetNamespaceAccess(ctx.Request.Context(), ctx.Param("id"), req.UserName, req.Access); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) setVolumeAccessHandler(ctx *gin.Context) {
	var req model.SetUserAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.SetVolumeAccess(ctx.Request.Context(), ctx.Param("id"), req.UserName, req.Access); err != nil {
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

func (ah *accessHandlers) deleteVolumeAccessHandler(ctx *gin.Context) {
	var req model.DeleteUserAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.DeleteVolumeAccess(ctx.Request.Context(), ctx.Param("id"), req.UserName); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) getNamespaceAccessHandler(ctx *gin.Context) {
	ret, err := ah.acts.GetNamespaceAccess(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	httputil.MaskForNonAdmin(ctx, &ret)
	ctx.JSON(http.StatusOK, ret)
}

func (ah *accessHandlers) getVolumeAccessHandler(ctx *gin.Context) {
	ret, err := ah.acts.GetVolumeAccess(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	httputil.MaskForNonAdmin(ctx, &ret)
	ctx.JSON(http.StatusOK, ret)
}

func (r *Router) SetupAccessRoutes(acts server.AccessActions) {
	handlers := &accessHandlers{acts: acts, tv: r.tv}

	// swagger:operation GET /access Permissions GetResourcesAccesses
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
	//     description: user accesses to resources
	//     schema:
	//       $ref: '#/definitions/ResourcesAccesses'
	//	 default:
	//	   $ref: '#/responses/error'
	r.engine.GET("/access", handlers.getUserAccessesHandler)

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

	// swagger:operation PUT /namespaces/{id}/access Permissions SetNamespaceAccess
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
	r.engine.PUT("/namespaces/:id/access", handlers.setNamespaceAccessHandler)

	// swagger:operation PUT /volumes/{id}/access Permissions SetVolumeAccess
	//
	// Grant volume permission to user.
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
	//  - name: id
	//    in: path
	//    required: true
	//    description: Volume ID
	// responses:
	//	 '200':
	//	   description: access set
	//	 default:
	//	   $ref: '#/responses/error'
	r.engine.PUT("/volumes/:id/access", handlers.setVolumeAccessHandler)

	// swagger:operation DELETE /namespaces/{id}/access Permissions DeleteNamespaceAccess
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
	r.engine.DELETE("/namespaces/:id/access", handlers.deleteNamespaceAccessHandler)

	// swagger:operation DELETE /volumes/{id}/access Permissions DeleteVolumeAccess
	//
	// Delete volume permission to user.
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
	r.engine.DELETE("/volumes/:id/accesses", handlers.deleteVolumeAccessHandler)

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
	//       $ref: '#/definitions/NamespaceWithPermissions'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/namespaces/:id/accesses", handlers.getNamespaceAccessHandler)

	// swagger:operation GET /volumes/{id}/accesses Permissions GetVolumeWithPermissions
	//
	// Get volume with user permissions.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ResourceID'
	// responses:
	//   '200':
	//     description: volume response
	//     schema:
	//       $ref: '#/definitions/VolumeWithPermissions'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/volumes/:id/accesses", handlers.getVolumeAccessHandler)
}
