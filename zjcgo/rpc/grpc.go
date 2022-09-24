package rpc

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

//listen, _ := net.Listen("tcp", ":9111")
//server := grpc.NewServer()
//api.RegisterGoodsApiServer(server, &api.GoodsRpcService{})
//err := server.Serve(listen)
//log.Println(err)
type ZjcrpcServer struct {
	liseten  net.Listener
	g        *grpc.Server
	register []func(g *grpc.Server)
	ops      []grpc.ServerOption
}

func NewGrpcServer(addr string, ops ...ZjcrpcOption) (*ZjcrpcServer, error) {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	zjc := &ZjcrpcServer{}
	zjc.liseten = listen
	for _, v := range ops {
		v.Apply(zjc)
	}
	server := grpc.NewServer(zjc.ops...)
	zjc.g = server
	return zjc, nil
}
func (s *ZjcrpcServer) Stop() {
	s.g.Stop()
}
func (s *ZjcrpcServer) Run() error {
	for _, f := range s.register {
		f(s.g)
	}
	return s.g.Serve(s.liseten)
}

func (s *ZjcrpcServer) Register(f func(g *grpc.Server)) {
	s.register = append(s.register, f)
}

type ZjcrpcOption interface {
	Apply(s *ZjcrpcServer)
}
type DefaultZjcGrpcOption struct {
	f func(s *ZjcrpcServer)
}

func (d DefaultZjcGrpcOption) Apply(s *ZjcrpcServer) {
	d.f(s)
}
func withGrpcOptions(ops ...grpc.ServerOption) ZjcrpcOption {
	return &DefaultZjcGrpcOption{
		f: func(s *ZjcrpcServer) {
			s.ops = append(s.ops, ops...)
		},
	}
}

type ZjcrpcClient struct {
	Conn *grpc.ClientConn
}

func NewGrpcClient(config *ZjcrpcClientConfig) (*ZjcrpcClient, error) {
	var ctx = context.Background()
	var dialOptions = config.dialOptions

	if config.Block {
		//阻塞
		if config.DialTimeout > time.Duration(0) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.DialTimeout)
			defer cancel()
		}
		dialOptions = append(dialOptions, grpc.WithBlock())
	}
	if config.KeepAlive != nil {
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(*config.KeepAlive))
	}
	conn, err := grpc.DialContext(ctx, config.Address, dialOptions...)
	if err != nil {
		return nil, err
	}
	return &ZjcrpcClient{
		Conn: conn,
	}, nil
}

type ZjcrpcClientConfig struct {
	Address     string
	Block       bool
	DialTimeout time.Duration
	ReadTimeout time.Duration
	Direct      bool
	KeepAlive   *keepalive.ClientParameters
	dialOptions []grpc.DialOption
}

func DefaultGrpcClientConfig() *ZjcrpcClientConfig {
	return &ZjcrpcClientConfig{
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		DialTimeout: time.Second * 3,
		ReadTimeout: time.Second * 2,
		Block:       true,
	}
}
