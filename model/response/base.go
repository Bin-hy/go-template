package response

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response[T any] struct {
    Code int    `json:"code"`
    Msg  string `json:"msg"`
    Data T      `json:"data,omitempty"`
}

func JSON[T any](c *gin.Context, status int, code int, msg string, data T) {
    c.JSON(status, Response[T]{
        Code: code,
        Msg:  msg,
        Data: data,
    })
}

func Success[T any](c *gin.Context, data T) {
    JSON(c, http.StatusOK, 0, "success", data)
}

func Error(c *gin.Context, status int, code int, msg string) {
    c.JSON(status, Response[any]{
        Code: code,
        Msg:  msg,
        Data: nil,
    })
}