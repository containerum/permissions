package router

import (
	"net/http"

	"git.containerum.net/ch/permissions/pkg/model"
	"git.containerum.net/ch/permissions/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type volumeHandlers struct {
	tv   *TranslateValidate
	acts server.VolumeActions
}

func (vh *volumeHandlers) createVolumeHandler(ctx *gin.Context) {
	var req model.VolumeCreateRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.BadRequest(ctx, err))
		return
	}
	if err := vh.acts.CreateVolume(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (r *Router) SetupVolumeHandlers(acts server.VolumeActions) {
	handlers := &volumeHandlers{tv: r.tv, acts: acts}

	// swagger:operation POST /volumes Volumes CreateVolume
	//
	// Create Volume for User by Tariff.
	// Should be chosen first storage, where free space allows to create volume with provided capacity.
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
	//      $ref: '#/definitions/VolumeCreateRequest'
	// responses:
	//   '201':
	//     description: volume created
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/volumes", handlers.createVolumeHandler)
}
