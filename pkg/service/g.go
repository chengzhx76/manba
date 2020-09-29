package service

import (
	"manba/pkg/pb/rpcpb"
	"manba/pkg/store"
)

var (
	// MetaService global service
	MetaService rpcpb.MetaServiceServer
	// Store global store db
	Store store.Store
)

// Init init service package
func Init(db store.Store) {
	Store = db
	// 服务端实例 在创建 grpc 时需要传此对象
	MetaService = newMetaService(db)
}
