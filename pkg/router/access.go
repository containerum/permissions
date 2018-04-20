package router

import (
	"net/http"

	"git.containerum.net/ch/permissions/pkg/server"
	"git.containerum.net/ch/utils/httputil"
	"github.com/gin-gonic/gin"
)

type accessHandlers struct {
	tv   *TranslateValidate
	acts server.AccessActions
}

func (ah *accessHandlers) getUserAccessesHandler(ctx *gin.Context) {
	userID := httputil.MustGetUserID(ctx)
	ret, err := ah.acts.GetUserAccesses(ctx.Request.Context(), userID)
	if err != nil {
		ctx.AbortWithStatusJSON(ah.tv.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, ret)
}

func (r *Router) SetupAccessRoutes(acts server.AccessActions) {
	handlers := &accessHandlers{acts: acts, tv: r.tv}

	r.engine.GET("/access", handlers.getUserAccessesHandler)
}
