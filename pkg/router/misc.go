package router

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func getFilters(ctx *gin.Context) []string {
	q := ctx.Query("filter")
	if len(q) == 0 {
		return nil
	}
	return strings.Split(q, ",")
}

func getPaginationParams(ctx *gin.Context) (page, perPage int, err error) {
	page, err = strconv.Atoi(ctx.Query("page"))
	if err != nil {
		err = fmt.Errorf("page number not integer")
		return
	}
	perPage, err = strconv.Atoi(ctx.Query("per_page"))
	if err != nil {
		err = fmt.Errorf("per page limit not integer")
		return
	}
	return
}
