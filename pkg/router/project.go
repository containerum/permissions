package router

import (
	"net/http"

	"git.containerum.net/ch/permissions/pkg/model"
	"git.containerum.net/ch/permissions/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type projectHandlers struct {
	tv   *TranslateValidate
	acts server.ProjectActions
}

func (ph *projectHandlers) createProject(ctx *gin.Context) {
	var req model.ProjectCreateRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.BadRequest(ctx, err))
		return
	}

	if err := ph.acts.CreateProject(ctx.Request.Context(), req.Label); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (r *Router) SetupProjectHandlers(acts server.ProjectActions) {
	handlers := &projectHandlers{tv: r.tv, acts: acts}

	// swagger:operation POST /projects Projects CreateProject
	//
	// Create project.
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/ProjectCreateRequest'
	// responses:
	//   '201':
	//     description: project created
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/projects", handlers.createProject)
}
