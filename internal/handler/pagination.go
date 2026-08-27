package handler

import "github.com/gin-gonic/gin"

func hasPaginationQuery(c *gin.Context) bool {
	return c.Query("page") != "" || c.Query("limit") != "" || c.Query("page_size") != ""
}
