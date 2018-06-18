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

type projectHandlers struct {
	tv   *TranslateValidate
	acts server.ProjectActions
}

func (ph *projectHandlers) createProjectHandler(ctx *gin.Context) {
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

func (ph *projectHandlers) addGroupToProjectHandler(ctx *gin.Context) {
	var req model.ProjectAddGroupRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.BadRequest(ctx, err))
		return
	}

	if err := ph.acts.AddGroup(ctx.Request.Context(), ctx.Param("project"), req.GroupID); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

func (ph *projectHandlers) getProjectGroupsHandler(ctx *gin.Context) {
	groups, err := ph.acts.GetProjectGroups(ctx.Request.Context(), ctx.Param("project"))
	if err != nil {
		ctx.AbortWithStatusJSON(ph.tv.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (r *Router) SetupProjectRoutes(acts server.ProjectActions) {
	handlers := &projectHandlers{tv: r.tv, acts: acts}

	// swagger:operation POST /projects Projects CreateProject
	//
	// Create project.
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
	//      $ref: '#/definitions/ProjectCreateRequest'
	// responses:
	//   '201':
	//     description: project created
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/projects", handlers.createProjectHandler)

	// swagger:operation POST /projects/{project}/groups Projects AddGroupToProject
	//
	// Add group to project (admin only).
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: project
	//    in: path
	//    required: true
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/ProjectAddGroupRequest'
	// responses:
	//   '202':
	//     description: group added to project
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/projects/:project/groups", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.addGroupToProjectHandler)

	// swagger:operation GET /projects/{project}/groups Projects GetProjectGroups
	//
	// Get project groups.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - name: project
	//    in: path
	//    required: true
	// responses:
	//   '200':
	//     description: project groups
	//     schema:
	//       type: object
	//       properties:
	//         groups:
	//           type: array
	//           items:
	//             $ref: '#/definitions/UserGroup'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/projects/:project/groups", handlers.getProjectGroupsHandler)
}
