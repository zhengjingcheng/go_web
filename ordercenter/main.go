package main

import (
	"context"
	"github.com/zhengjingcheng/zjcgo"
	"github.com/zhengjingcheng/zjcgo/rpc"
	"github.com/zjc/ordercenter/service"
	"log"
	"net/http"
)

func main() {
	engine := zjcgo.Default()
	client := rpc.NewHttpClient()
	client.RegisterHttpService("goods", &service.GoodsService{})
	group := engine.Group("order")
	group.Get("/findTcp", func(ctx *zjcgo.Context) {
		//连接grpc服务
		proxy := rpc.NewMsTcpClientProxy(rpc.DefaultOption)
		params := make([]any, 1)
		params[0] = int64(1)
		result, err := proxy.Call(context.Background(), "goods", "Find", params)
		log.Println(err)
		ctx.JSON(http.StatusOK, result)
	})
	engine.Run(":9003")
}
