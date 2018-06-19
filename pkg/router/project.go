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

func (ph *projectHandlers) setGroupMemberAccessHandler(ctx *gin.Context) {
	var req model.SetGroupMemberAccessRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.BadRequest(ctx, err))
		return
	}

	if err := ph.acts.SetGroupMemberAccess(ctx.Request.Context(), ctx.Param("project"), ctx.Param("group"), req); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

func (ph *projectHandlers) deleteGroupFromProjectHandler(ctx *gin.Context) {
	if err := ph.acts.DeleteGroupFromProject(ctx.Request.Context(), ctx.Param("project"), ctx.Param("group")); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

func (ph *projectHandlers) addMemberToProjectHandler(ctx *gin.Context) {
	var req model.AddMemberToProjectRequest

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.BadRequest(ctx, err))
		return
	}

	if err := ph.acts.AddMemberToProject(ctx.Request.Context(), ctx.Param("project"), req); err != nil {
		ctx.AbortWithStatusJSON(ph.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
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
	//  - $ref: '#/parameters/ProjectID'
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
	//  - $ref: '#/parameters/ProjectID'
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

	// swagger:operation PUT /projects/{project}/groups/{group} Projects SetGroupMemberAccess
	//
	// Change access of group member to project namespaces.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/ProjectID'
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
	r.engine.PUT("/project/:project/groups/:group", handlers.setGroupMemberAccessHandler)

	// swagger:operation DELETE /projects/{project}/groups/{group} Projects DeleteGroupFromProject
	//
	// Delete group permissions from project namespaces.
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
	r.engine.DELETE("/project/:project/groups/:group", handlers.deleteGroupFromProjectHandler)

	// swagger:operation POST /projects/{project}/members Projects AddMemberToProject
	//
	// Add permissions for user to project namespaces.
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
	//      $ref: '#/definitions/AddMemberToProjectRequest'
	// responses:
	//   '202':
	//     description: member added
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/project/:project/members", handlers.addMemberToProjectHandler)
}
