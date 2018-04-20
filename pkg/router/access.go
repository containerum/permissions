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

func (r *Router) SetupAccessRoutes(acts server.AccessActions) {
	handlers := &accessHandlers{acts: acts, tv: r.tv}

	// swagger:operation GET /access GetResourcesAccesses
	//
	// Returns user accesses to resources
	//
	// ---
	// parameters:
	//	- $ref: '#/parameters/UserIDHeader'
	//	- $ref: '#/parameters/UserRoleHeader'
	//	- $ref: '#/parameters/SubstitutedUserID'
	// responses:
	//	'200':
	//	  description: accesses response
	//	default:
	//	  description: error
	r.engine.GET("/access", handlers.getUserAccessesHandler)

	// swagger:operation PUT /admin/accesses SetResourcesAccesses
	//
	// Assign access level for all user resources. Used for billing purposes.
	//
	// ---
	// parameters:
	// - $ref: '#/parameters/UserIDHeader'
	// - $ref: '#/parameters/UserRoleHeader'
	// - $ref: '#/parameters/SubstitutedUserID'
	// - name: body
	//   in: body
	//   schema:
	//     $ref: "#/definitions/SetResourcesAccessesRequest"
	// responses:
	//	'200':
	//	  description: access set
	//	default:
	//	  description: error
	r.engine.PUT("/admin/accesses", handlers.setUserAccessesHandler)
}
