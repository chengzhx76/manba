package main

import (
	"flag"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/labstack/echo"
	"manba/pkg/util"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/fagongzi/log"
	"google.golang.org/grpc"
	"manba/grpcx"
	"manba/pkg/pb/rpcpb"
	"manba/pkg/service"
	"manba/pkg/store"
)

var (
	addr           = flag.String("addr", "127.0.0.1:9092", "Addr: client grpc entrypoint")
	addrHTTP       = flag.String("addr-http", "127.0.0.1:9093", "Addr: client http restful entrypoint")
	addrStore      = flag.String("addr-store", "etcd://127.0.0.1:2379", "Addr: store address")
	addrStoreUser  = flag.String("addr-store-user", "", "addr Store UserName")
	addrStorePwd   = flag.String("addr-store-pwd", "", "addr Store Password")
	namespace      = flag.String("namespace", "dev", "The namespace to isolation the environment.")
	discovery      = flag.Bool("discovery", false, "Publish apiserver service via discovery.")
	servicePrefix  = flag.String("service-prefix", "/services", "The prefix for service name.")
	publishLease   = flag.Int64("publish-lease", 10, "Publish service lease seconds")
	publishTimeout = flag.Int("publish-timeout", 30, "Publish service timeout seconds")
	ui             = flag.String("ui", "/app/manba/ui/dist", "The gateway ui dist dir.")
	uiPrefix       = flag.String("ui-prefix", "/ui", "The gateway ui prefix path.")
	version        = flag.Bool("version", false, "Show version info")
)

// --addr=127.0.0.1:9091 --addr-store=etcd://180.76.183.68:2379 --discovery --namespace=test
// https://www.jianshu.com/p/431abe0d2ed5

func main() {
	flag.Parse()

	if *version {
		util.PrintVersion()
		os.Exit(0)
	}

	log.InitLog()
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Infof("addr: %s", *addr)                     // 请求地址
	log.Infof("addr-store: %s", *addrStore)          // etcd 地址
	log.Infof("addr-store-user: %s", *addrStoreUser) // etcd 用户
	log.Infof("addr-store-pwd: %s", *addrStorePwd)   // etcd 密码
	log.Infof("namespace: %s", *namespace)           // etcd 命名空间
	log.Infof("discovery: %v", *discovery)
	log.Infof("service-prefix: %s", *servicePrefix)
	log.Infof("publish-lease: %d", *publishLease)
	log.Infof("publish-timeout: %d", *publishTimeout)

	// 初始化DB
	db, err := store.GetStoreFrom(*addrStore, fmt.Sprintf("/%s", *namespace), *addrStoreUser, *addrStorePwd)
	if err != nil {
		log.Fatalf("init store failed for %s, errors:\n%+v", *addrStore, err)
	}

	// 服务关联db
	service.Init(db)

	var opts []grpcx.ServerOption
	if *discovery {
		dbClient := db.Raw().(*clientv3.Client)
		// 准备 etcdPublisher 参数
		etcdPublisher := grpcx.WithEtcdPublisher(dbClient, *servicePrefix, *publishLease, time.Second*time.Duration(*publishTimeout))
		opts = append(opts, etcdPublisher)
	}

	if *addrHTTP != "" {
		initHttpRouterFunc := func(server *echo.Echo) {
			// 初始化路由
			service.InitHTTPRouter(server, *ui, *uiPrefix)
		}
		// 准备 http 参数
		httpServer := grpcx.WithHTTPServer(*addrHTTP, initHttpRouterFunc)
		opts = append(opts, httpServer)
	}

	// 注册一个服务（对外提供的服务）
	registerServices := func(grpcServer *grpc.Server) []grpcx.Service {
		var services []grpcx.Service
		// Server端注册 rpc 服务
		rpcpb.RegisterMetaServiceServer(grpcServer, service.MetaService)
		_service := grpcx.NewService(rpcpb.ServiceMeta, nil)
		services = append(services, _service)
		return services
	}

	// 注册 grpc 服务
	grpcServer := grpcx.NewGRPCServer(*addr, registerServices, opts...)

	log.Infof("api server listen at %s", *addr)
	go grpcServer.Start()

	waitStop(grpcServer)
}

/*
 GRPC 的步骤
	1.设置监听端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 9000))
	2.提供服务的对象
	server := chat.Server{}
	3.初始 grpc 服务
	grpcServer := grpc.NewServer()
	4.注册业务服务到 grpc 上
	chat.RegisterChatServiceServer(grpcServer, &server)
	5.端口和 grpc 服务绑定
	err := grpcServer.Serve(lis)
*/

func waitStop(s *grpcx.GRPCServer) {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-sc
	s.GracefulStop()
	log.Infof("exit: signal=<%d>.", sig)
	switch sig {
	case syscall.SIGTERM:
		log.Infof("exit: bye :-).")
		os.Exit(0)
	default:
		log.Infof("exit: bye :-(.")
		os.Exit(1)
	}
}
