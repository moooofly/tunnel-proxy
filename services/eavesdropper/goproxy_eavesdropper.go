package eavesdropper

import (
	"bufio"
	logger "log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/moooofly/http-tunnel/services"
)

type EavesdropperArgs struct {
	Local *string
}

type Eavesdropper struct {
	cfg    EavesdropperArgs
	log    *logger.Logger
	isStop bool
}

func NewEavesdropper() services.Service {
	return &Eavesdropper{
		cfg:    EavesdropperArgs{},
		isStop: false,
	}
}

func (s *Eavesdropper) StopService() {
	defer func() {
		e := recover()
		if e != nil {
			s.log.Printf("stop eavesdropper proxy crashed,%s", e)
		} else {
			s.log.Printf("service eavesdropper proxy stopped")
		}
		s.cfg = EavesdropperArgs{}
		s.log = nil
		s = nil
	}()
	s.isStop = true
}

func (s *Eavesdropper) Start(args interface{}, log *logger.Logger) (err error) {
	s.log = log
	s.cfg = args.(EavesdropperArgs)

	for _, addr := range strings.Split(*s.cfg.Local, ",") {
		if addr != "" {

			// ref: https://github.com/elazarl/goproxy/blob/master/examples/goproxy-eavesdropper/main.go

			proxy := goproxy.NewProxyHttpServer()
			proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*baidu.com$"))).
				HandleConnect(goproxy.AlwaysReject)
			proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
				HandleConnect(goproxy.AlwaysMitm)
				// enable curl -p for all hosts on port 80
			proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*:80$"))).
				HijackConnect(func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
					defer func() {
						if e := recover(); e != nil {
							ctx.Logf("error connecting to remote: %v", e)
							client.Write([]byte("HTTP/1.1 500 Cannot reach destination\r\n\r\n"))
						}
						client.Close()
					}()
					clientBuf := bufio.NewReadWriter(bufio.NewReader(client), bufio.NewWriter(client))
					remote, err := net.Dial("tcp", req.URL.Host)
					if err != nil {
						return
					}
					remoteBuf := bufio.NewReadWriter(bufio.NewReader(remote), bufio.NewWriter(remote))
					for {
						req, err := http.ReadRequest(clientBuf.Reader)
						if err != nil {
							return
						}
						if err = req.Write(remoteBuf); err != nil {
							return
						}
						if err = remoteBuf.Flush(); err != nil {
							return
						}
						resp, err := http.ReadResponse(remoteBuf.Reader, req)
						if err != nil {
							return
						}
						if err = resp.Write(clientBuf.Writer); err != nil {
							return
						}
						if err = clientBuf.Flush(); err != nil {
							return
						}
					}
				})
			//proxy.Verbose = *verbose
			err = http.ListenAndServe(addr, proxy)
		}
	}
	return
}

func (s *Eavesdropper) Clean() {
	s.StopService()
}
