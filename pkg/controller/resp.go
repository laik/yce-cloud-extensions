package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type message struct {
	Data string `json:"data"`
	Msg  string `json:"msg"`
}

func requestErr(g *gin.Context, err error) {
	g.JSON(http.StatusOK, &message{Data: err.Error(), Msg: "request not match"})
	g.Abort()
}
