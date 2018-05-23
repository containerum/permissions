package router

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func getFilters(values url.Values) []string {
	q := values.Get("filter")
	if len(q) == 0 {
		return nil
	}
	return strings.Split(q, ",")
}

func getPaginationParams(values url.Values) (page, perPage int, err error) {
	if values.Get("per_page") == "" {
		return 0, 0, nil
	}
	page, err = strconv.Atoi(values.Get("page"))
	if err != nil {
		err = fmt.Errorf("page number not integer")
		return
	}
	perPage, err = strconv.Atoi(values.Get("per_page"))
	if err != nil {
		err = fmt.Errorf("per page limit not integer")
		return
	}
	return
}
