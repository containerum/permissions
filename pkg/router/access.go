package router

import (
	"net/http"

	"git.containerum.net/ch/permissions/pkg/model"
	"git.containerum.net/ch/permissions/pkg/server"
	"git.containerum.net/ch/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type accessHandlers struct {
	tv   *TranslateValidate
	acts server.AccessActions
}

func (ah *accessHandlers) getUserAccessesHandler(ctx *gin.Context) {
	userID := httputil.MustGetUserID(ctx.Request.Context())
	ret, err := ah.acts.GetUserAccesses(ctx.Request.Context(), userID)
	if err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, ret)
}

func (ah *accessHandlers) setUserAccessesHandler(ctx *gin.Context) {
	userID := httputil.MustGetUserID(ctx.Request.Context())
	var req model.SetUserAccessesRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.SetUserAccesses(ctx.Request.Context(), userID, req.Access); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) setNamespaceAccessHandler(ctx *gin.Context) {
	userID := httputil.MustGetUserID(ctx.Request.Context())
	var req model.SetUserAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.SetNamespaceAccess(ctx, userID, ctx.Param("label"), req.UserName, req.Access); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) setVolumeAccessHandler(ctx *gin.Context) {
	userID := httputil.MustGetUserID(ctx.Request.Context())
	var req model.SetUserAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.SetVolumeAccess(ctx, userID, ctx.Param("label"), req.UserName, req.Access); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) deleteNamespaceAccessHandler(ctx *gin.Context) {
	userID := httputil.MustGetUserID(ctx.Request.Context())
	var req model.DeleteUserAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.DeleteNamespaceAccess(ctx, userID, ctx.Param("label"), req.UserName); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (ah *accessHandlers) deleteVolumeAccessHandler(ctx *gin.Context) {
	userID := httputil.MustGetUserID(ctx.Request.Context())
	var req model.DeleteUserAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.BadRequest(ctx, err))
		return
	}

	if err := ah.acts.DeleteVolumeAccess(ctx, userID, ctx.Param("label"), req.UserName); err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (r *Router) SetupAccessRoutes(acts server.AccessActions) {
	handlers := &accessHandlers{acts: acts, tv: r.tv}

	// swagger:operation GET /access GetResourcesAccesses
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
	//	   description: error
	r.engine.GET("/access", handlers.getUserAccessesHandler)

	// swagger:operation PUT /admin/accesses SetResourcesAccesses
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
	//	'200':
	//	  description: access set
	//	default:
	//	  description: error
	r.engine.PUT("/admin/accesses", handlers.setUserAccessesHandler)

	// swagger:operation PUT /namespaces/{label}/access SetNamespaceAccess
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
	//  - name: label
	//    in: path
	//    required: true
	//    description: Namespace label
	// responses:
	//	'200':
	//	  description: access set
	//	default:
	//	  description: error
	r.engine.PUT("/namespaces/:label/access", handlers.setNamespaceAccessHandler)

	// swagger:operation PUT /volumes/{label}/access SetVolumeAccess
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
	//  - name: label
	//    in: path
	//    required: true
	//    description: Volume label
	// responses:
	//	'200':
	//	  description: access set
	//	default:
	//	  description: error
	r.engine.PUT("/volumes/:label/access", handlers.setVolumeAccessHandler)

	// swagger:operation DELETE /namespaces/{label}/access DeleteNamespaceAccess
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
	//  - name: label
	//    in: path
	//    required: true
	//    description: Namespace label
	// responses:
	//	'200':
	//	  description: access deleted
	//	default:
	//	  description: error
	r.engine.DELETE("/namespaces/:label/access", handlers.deleteNamespaceAccessHandler)

	// swagger:operation DELETE /volumes/{label}/access DeleteVolumeAccess
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
	//  - name: label
	//    in: path
	//    required: true
	//    description: Namespace label
	// responses:
	//	'200':
	//	  description: access deleted
	//	default:
	//	  description: error
	r.engine.DELETE("/volumes/:label/accesses", handlers.deleteVolumeAccessHandler)
}
