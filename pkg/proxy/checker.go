package proxy

import (
	"context"
	"time"

	"github.com/valyala/fasthttp"
	"manba/log"
	"manba/pkg/pb/metapb"
	"manba/pkg/store"
	"manba/pkg/util"
)

// 开始心跳检查
func (r *dispatcher) readyToHeathChecker() {
	for i := 0; i < r.cnf.Option.LimitCountHeathCheckWorker; i++ {
		r.runner.RunCancelableTask(func(ctx context.Context) {
			log.Infof("start server check worker")

			for {
				select {
				case <-ctx.Done():
					return
				case id := <-r.checkerC:
					r.check(id)
				}
			}
		})
	}
}

func (r *dispatcher) addToCheck(svr *serverRuntime) {
	svr.circuit = metapb.Open // 默认正常状态
	if svr.meta.HeathCheck != nil {
		svr.useCheckDuration = time.Duration(svr.meta.HeathCheck.CheckInterval)
	}
	svr.heathTimeout.Stop()
	r.checkerC <- svr.meta.ID
}

func (r *dispatcher) heathCheckTimeout(arg interface{}) {
	id := arg.(uint64)
	if _, ok := r.servers[id]; ok {
		r.checkerC <- id
	}
}

func (r *dispatcher) check(id uint64) {
	svr, ok := r.servers[id]
	if !ok {
		return
	}

	defer func() {
		if svr.meta.HeathCheck != nil {
			// 健康检查间隔 服务平台配置的大于本地配置的用本地的配置
			if svr.useCheckDuration > r.cnf.Option.LimitIntervalHeathCheck {
				svr.useCheckDuration = r.cnf.Option.LimitIntervalHeathCheck
			}

			// 健康检查间隔如果是 0，则用服务端配置的
			if svr.useCheckDuration == 0 {
				svr.useCheckDuration = time.Duration(svr.meta.HeathCheck.CheckInterval)
			}

			// 间隔时间内调用 r.heathCheckTimeout 方法
			svr.heathTimeout, _ = r.tw.Schedule(svr.useCheckDuration, r.heathCheckTimeout, id)
		}
	}()

	status := metapb.Unknown
	prev := r.getServerStatus(svr.meta.ID)

	// 没有设置健康检查逻辑
	if svr.meta.HeathCheck == nil {
		log.Warnf("server <%d> heath check not setting", svr.meta.ID)
		r.watchEventC <- &store.Evt{
			Src:  eventSrcStatusChanged,
			Type: eventTypeStatusChanged,
			Value: statusChanged{
				meta:   *svr.meta,
				status: metapb.Up, // 默认为可用
			},
		}
		return
	}

	// 检查状态
	if r.doCheck(svr) {
		status = metapb.Up
	} else {
		status = metapb.Down
	}

	log.Infof("===check==========> [ID:%d]-[%v]-[%v]", id, prev, status)

	// 上一个状态不等于现在的检查状态
	if prev != status {
		r.watchEventC <- &store.Evt{
			Src:  eventSrcStatusChanged,
			Type: eventTypeStatusChanged,
			Value: statusChanged{
				meta:   *svr.meta,
				status: status,
			},
		}
	}
}

// 检查服务器真正逻辑
func (r *dispatcher) doCheck(svr *serverRuntime) bool {

	log.Infof("server <%d, %s> start check", svr.meta.ID, svr.getCheckURL())

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(svr.getCheckURL())

	opt := util.DefaultHTTPOption()
	*opt = *globalHTTPOptions
	opt.ReadTimeout = time.Duration(svr.meta.HeathCheck.Timeout)

	resp, err := r.httpClient.Do(req, svr.meta.Addr, opt)
	defer fasthttp.ReleaseResponse(resp)
	if err != nil {
		log.Warnf("server <%d, %s, %d> check failed, errors:%+v", svr.meta.ID, svr.getCheckURL(), svr.checkFailCount+1, err)
		svr.fail()
		return false
	}

	if fasthttp.StatusOK != resp.StatusCode() {
		log.Warnf("server <%d, %s, %d, %d> check failed", svr.meta.ID, svr.getCheckURL(), resp.StatusCode(), svr.checkFailCount+1)
		svr.fail()
		return false
	}

	if svr.meta.HeathCheck.Body != "" && svr.meta.HeathCheck.Body != string(resp.Body()) {
		log.Warnf("server <%s, %s, %d> check failed, body <%s>, expect <%s>", svr.meta.Addr, svr.getCheckURL(), svr.checkFailCount+1, resp.Body(), svr.meta.HeathCheck.Body)
		svr.fail()
		return false
	}

	svr.reset()
	return true
}
