package service

import "github.com/zhengjingcheng/zjcgo/rpc"

type GoodsService struct {
	Find func(args map[string]any) ([]byte, error) `zjcrpc:"GET,/goods/find"`
}

func (*GoodsService) Env() rpc.HttpConfig {
	return rpc.HttpConfig{
		Host: "localhost",
		Port: 9002,
	}
}
