package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Detail  string      `json:"detail,omitempty"`
}

type PageResponse struct {
	Items interface{} `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Pages int         `json:"pages"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

func SuccessPage(c *gin.Context, data interface{}, total int, page int, size int, pages int) {
	c.JSON(http.StatusOK, PageResponse{
		Items: data,
		Total: total,
		Page:  page,
		Size:  size,
		Pages: pages,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Detail: message,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
