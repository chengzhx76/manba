/software/etcd/etcd-v3.4.12-linux-amd64
nohup ./etcd --listen-client-urls http://0.0.0.0:2379 --advertise-client-urls http://0.0.0.0:2379 &

/software/golang/src/manba/dist
nohup ./manba-proxy --addr=0.0.0.0:8076 --addr-rpc=0.0.0.0:9091 --addr-store=etcd://180.76.183.68:2379 --namespace=test &

nohup ./manba-apiserver --addr-http=0.0.0.0:9093 --addr=0.0.0.0:9091 --addr-store=etcd://180.76.183.68:2379 --discovery --namespace=test -ui=ui/dist >api-server.log &

nohup go run backend.go --addr=0.0.0.0:9000 >9000.log &
nohup go run backend.go --addr=0.0.0.0:9001 >9001.log &
nohup go run backend.go --addr=0.0.0.0:9002 >9002.log &



make ui

http://127.0.0.1:8076/v1/users/1



/software/etcd/etcd-v3.4.12-linux-amd64
nohup ./etcd --listen-client-urls http://0.0.0.0:2379 --advertise-client-urls http://0.0.0.0:2379 &

/software/golang/src/manba/dist
nohup ./manba-proxy --addr=0.0.0.0:8076 --addr-rpc=121.4.41.132:9091 --addr-store=etcd://127.0.0.1:2379 --namespace=test &

## 本地连接 
--addr=0.0.0.0:8076 --addr-rpc=121.4.41.132:9091 --addr-store=etcd://121.4.41.132:2379 --namespace=test

nohup ./manba-apiserver --addr-http=0.0.0.0:9093 --addr=0.0.0.0:9091 --addr-store=etcd://127.0.0.1:2379 --discovery --namespace=test -ui=ui/dist &

nohup go run backend.go --addr=0.0.0.0:9000 >9000.log &
nohup go run backend.go --addr=0.0.0.0:9001 >9001.log &
nohup go run backend.go --addr=0.0.0.0:9002 >9002.log &