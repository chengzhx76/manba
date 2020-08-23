package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"manba/pkg/util"

	"github.com/coreos/etcd/clientv3"
	"github.com/fagongzi/grpcx"
	"github.com/fagongzi/log"
	"github.com/labstack/echo"
	"google.golang.org/grpc"
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

func main() {
	flag.Parse()

	if *version {
		util.PrintVersion()
		os.Exit(0)
	}

	log.InitLog()
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Infof("addr: %s", *addr)
	log.Infof("addr-store: %s", *addrStore)
	log.Infof("addr-store-user: %s", *addrStoreUser)
	log.Infof("addr-store-pwd: %s", *addrStorePwd)
	log.Infof("namespace: %s", *namespace)
	log.Infof("discovery: %v", *discovery)
	log.Infof("service-prefix: %s", *servicePrefix)
	log.Infof("publish-lease: %d", *publishLease)
	log.Infof("publish-timeout: %d", *publishTimeout)

	db, err := store.GetStoreFrom(*addrStore, fmt.Sprintf("/%s", *namespace), *addrStoreUser, *addrStorePwd)
	if err != nil {
		log.Fatalf("init store failed for %s, errors:\n%+v",
			*addrStore,
			err)
	}

	service.Init(db)

	var opts []grpcx.ServerOption
	if *discovery {
		opts = append(opts, grpcx.WithEtcdPublisher(db.Raw().(*clientv3.Client), *servicePrefix, *publishLease, time.Second*time.Duration(*publishTimeout)))
	}

	if *addrHTTP != "" {
		opts = append(opts, grpcx.WithHTTPServer(*addrHTTP, func(server *echo.Echo) {
			service.InitHTTPRouter(server, *ui, *uiPrefix)
		}))
	}

	s := grpcx.NewGRPCServer(*addr, func(svr *grpc.Server) []grpcx.Service {
		var services []grpcx.Service
		rpcpb.RegisterMetaServiceServer(svr, service.MetaService)
		services = append(services, grpcx.NewService(rpcpb.ServiceMeta, nil))
		return services
	}, opts...)

	log.Infof("api server listen at %s", *addr)
	go s.Start()

	waitStop(s)
}

func waitStop(s *grpcx.GRPCServer) {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

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
