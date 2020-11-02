package proxy

import (
	"net"
	"net/http"
	"sync/atomic"

	"github.com/soheilhy/cmux"
	"github.com/valyala/fasthttp"
	"manba/log"
	"manba/pkg/util"
)

// Start start proxy
func (p *Proxy) Start() {
	go p.listenToStop()

	// 开始指标采集 推送到普罗米修斯
	p.startMetrics()
	// 开始任务
	p.startReadyTasks()

	if !p.cfg.Option.EnableWebSocket {
		go p.startHTTPS()
		p.startHTTP()

		return
	}

	go p.startHTTPSCMUX()
	p.startHTTPCMUX()
}

// Stop stop the proxy
func (p *Proxy) Stop() {
	log.Infof("stop: start to stop gateway proxy")

	p.stopWG.Add(1)
	p.stopC <- struct{}{}
	p.stopWG.Wait()

	log.Infof("stop: gateway proxy stopped")
}

func (p *Proxy) listenToStop() {
	<-p.stopC
	p.doStop()
}

func (p *Proxy) doStop() {
	p.stopOnce.Do(func() {
		defer p.stopWG.Done()
		p.setStopped()
		p.runner.Stop()
	})
}

func (p *Proxy) stopRPC() error {
	return p.rpcListener.Close()
}

func (p *Proxy) setStopped() {
	atomic.StoreInt32(&p.stopped, 1)
}

func (p *Proxy) isStopped() bool {
	return atomic.LoadInt32(&p.stopped) == 1
}

// 开始监控指标 推送
func (p *Proxy) startMetrics() {
	util.StartMetricsPush(p.runner, p.cfg.Metric)
}

func (p *Proxy) startReadyTasks() {
	// 插件里的资源 GC
	p.readyToGCJSEngine()
	// 请求复制
	p.readyToCopy()
	// Dispatch 监听
	p.readyToDispatch()
}

func (p *Proxy) newHTTPServer() *fasthttp.Server {
	return &fasthttp.Server{
		Handler:                       p.ServeFastHTTP,
		ReadBufferSize:                p.cfg.Option.LimitBufferRead,
		WriteBufferSize:               p.cfg.Option.LimitBufferWrite,
		MaxRequestBodySize:            p.cfg.Option.LimitBytesBody,
		DisableHeaderNamesNormalizing: p.cfg.Option.DisableHeaderNameNormalizing,
	}
}

func (p *Proxy) startHTTP() {
	log.Infof("start http at %s", p.cfg.Addr)
	htp := p.newHTTPServer()
	err := htp.ListenAndServe(p.cfg.Addr)
	if err != nil {
		log.Fatalf("start http listeners failed with %+v", err)
	}
}

func (p *Proxy) startHTTPWithListener(lis net.Listener) {
	log.Infof("start http at %s", p.cfg.Addr)
	htp := p.newHTTPServer()
	err := htp.Serve(lis)
	if err != nil {
		log.Fatalf("start http listeners failed with %+v", err)
	}
}

func (p *Proxy) startHTTPS() {
	if !p.enableHTTPS() {
		return
	}

	defaultCertData, defaultKeyData := p.mustParseDefaultTLSCert()

	log.Infof("start https at %s", p.cfg.AddrHTTPS)
	htsSever := p.newHTTPServer()
	p.appendCertsEmbed(htsSever, defaultCertData, defaultKeyData)
	err := htsSever.ListenAndServeTLS(p.cfg.AddrHTTPS, "", "")
	if err != nil {
		log.Fatalf("start http listeners failed with %+v", err)
	}
}

func (p *Proxy) startHTTPSWithListener(lis net.Listener) {
	defaultCertData, defaultKeyData := p.mustParseDefaultTLSCert()

	log.Infof("start https at %s", p.cfg.AddrHTTPS)
	htp := p.newHTTPServer()
	p.appendCertsEmbed(htp, defaultCertData, defaultKeyData)
	err := htp.ServeTLS(lis, "", "")
	if err != nil {
		log.Fatalf("start http listeners failed with %+v", err)
	}
}

func (p *Proxy) startHTTPWebSocketWithListener(lis net.Listener) {
	log.Infof("start http websocket at %s", p.cfg.Addr)
	s := &http.Server{
		Handler: p,
	}
	err := s.Serve(lis)
	if err != nil {
		log.Fatalf("start http websocket failed with %+v", err)
	}
}

func (p *Proxy) startHTTPSWebSocketWithListener(lis net.Listener) {
	defaultCertData, defaultKeyData := p.mustParseDefaultTLSCert()

	log.Infof("start https websocket at %s", p.cfg.Addr)
	s := &http.Server{
		Handler: p,
	}
	p.configTLSConfig(s, defaultCertData, defaultKeyData)
	err := s.ServeTLS(lis, "", "")
	if err != nil {
		log.Fatalf("start https websocket failed with errors %+v", err)
	}
}

func (p *Proxy) startHTTPCMUX() {
	lis, err := net.Listen("tcp", p.cfg.Addr)
	if err != nil {
		log.Fatalf("start http failed failed with %+v", err)
	}

	cm := cmux.New(lis)
	go p.startHTTPWithListener(cm.Match(cmux.Any()))
	go p.startHTTPWebSocketWithListener(cm.Match(cmux.HTTP1HeaderField("Upgrade", "websocket")))
	err = cm.Serve()
	if err != nil {
		log.Fatalf("start http failed failed with %+v", err)
	}
}

func (p *Proxy) startHTTPSCMUX() {
	if !p.enableHTTPS() {
		return
	}

	lis, err := net.Listen("tcp", p.cfg.AddrHTTPS)
	if err != nil {
		log.Fatalf("start https failed failed with %+v", err)
	}

	cm := cmux.New(lis)
	go p.startHTTPSWithListener(cm.Match(cmux.Any()))
	go p.startHTTPSWebSocketWithListener(cm.Match(cmux.HTTP1HeaderField("Upgrade", "websocket")))
	err = cm.Serve()
	if err != nil {
		log.Fatalf("start https failed failed with %+v", err)
	}
}
