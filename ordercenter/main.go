package main

import (
	"github.com/zhengjingcheng/zjcgo"
	"github.com/zhengjingcheng/zjcgo/rpc"
	"log"
)

func main() {
	engine := zjcgo.Default()
	client := rpc.NewHttpClient()
	group := engine.Group("order")
	group.Get("/find", func(ctx *zjcgo.Context) {
		//通过商品中心，查询商品的信息
		//通过http的方式进行调用
		params := make(map[string]any)
		params["id"] = 1000
		body, err := client.PostJson("http://localhost:9002/goods/find", params)
		if err != nil {
			panic(err)
		}
		log.Println(string(body))

	})
	engine.Run(":9003")
}
