package main

import (
	"github.com/zhengjingcheng/zjcgo"
	"github.com/zjc/goodscenter/model"
	"net/http"
)

func main() {
	engine := zjcgo.Default()
	group := engine.Group("goods")
	group.Get("/find", func(ctx *zjcgo.Context) {
		goods := &model.Goods{Id: 1000, Name: "zjc"}
		ctx.JSON(http.StatusOK, &model.Result{Code: 200, Msg: "success", Data: goods})
	})
	group.Post("/find", func(ctx *zjcgo.Context) {
		goods := &model.Goods{Id: 1000, Name: "zjc"}
		ctx.JSON(http.StatusOK, &model.Result{Code: 200, Msg: "success", Data: goods})
	})
	engine.Run(":9002")
}
