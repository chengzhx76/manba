package grpcx

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	etcdnaming "github.com/coreos/etcd/clientv3/naming"
	"google.golang.org/grpc/naming"
)

type etcdPublisher struct {
	prefix   string
	ttl      int64
	timeout  time.Duration
	client   *clientv3.Client
	resolver *etcdnaming.GRPCResolver
}

func newEtcdPublisher(client *clientv3.Client, prefix string, ttl int64, timeout time.Duration) (Publisher, error) {
	return &etcdPublisher{
		prefix:  prefix,
		ttl:     ttl,
		timeout: timeout,
		client:  client,
		resolver: &etcdnaming.GRPCResolver{ // 创建命名解析
			Client: client,
		},
	}, nil
}

func (p *etcdPublisher) Publish(service string, meta naming.Update) error {
	// 申请租约
	lessor := clientv3.NewLease(p.client)
	defer lessor.Close()

	// 设置租约时间的超时时间
	ctx, cancel := context.WithTimeout(p.client.Ctx(), p.timeout)
	// 设置租约时间
	leaseResp, err := lessor.Grant(ctx, p.ttl)
	cancel()
	if err != nil {
		return err
	}

	// 设置续租 定期发送需求请求
	_, err = p.client.KeepAlive(p.client.Ctx(), leaseResp.ID)
	if err != nil {
		return err
	}

	ctx, cancel = context.WithTimeout(p.client.Ctx(), p.timeout)
	defer cancel()

	// etcd用于grpc命名解析与服务发现 https://blog.csdn.net/ys5773477/article/details/80216208
	// gRPC Name Resolver 原理及实践 https://xiaomi-info.github.io/2019/12/31/grpc-custom-ns/
	// http://morecrazy.github.io/2018/08/14/grpc-go%E5%9F%BA%E4%BA%8Eetcd%E5%AE%9E%E7%8E%B0%E6%9C%8D%E5%8A%A1%E5%8F%91%E7%8E%B0%E6%9C%BA%E5%88%B6/

	// 将本服务注册添加etcd中，服务名称为 %s/%s，服务地址为 meta 里的地址
	return p.resolver.Update(ctx, fmt.Sprintf("%s/%s", p.prefix, service), meta, clientv3.WithLease(leaseResp.ID))
}
