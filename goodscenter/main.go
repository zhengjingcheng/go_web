package main

import (
	"encoding/gob"
	"github.com/zhengjingcheng/zjcgo"
	"github.com/zhengjingcheng/zjcgo/rpc"
	"github.com/zjc/goodscenter/model"
	"github.com/zjc/goodscenter/service"
	"log"
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
	//grpc方式注册服务
	tcpServer, err := rpc.NewTcpServer(":9222")
	log.Println(err)
	gob.Register(&model.Result{})
	gob.Register(&model.Goods{})
	tcpServer.Register("goods", &service.GoodsRpcService{})
	tcpServer.Run()

	engine.Run(":9002")
}
