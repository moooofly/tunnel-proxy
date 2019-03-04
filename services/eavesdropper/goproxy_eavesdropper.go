package eavesdropper

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/moooofly/goproxy/services"
	"github.com/moooofly/goproxy/utils"
	"github.com/moooofly/goproxy/utils/authx"
	"github.com/sirupsen/logrus"
)

const realm = "EavesDropper-Realm"

type EavesdropperArgs struct {
	Local    *string
	White    *string
	AuthFile *string
	Auth     *[]string
}

type Eavesdropper struct {
	cfg       EavesdropperArgs
	basicAuth authx.BasicAuth
	log       *logrus.Logger
	isStop    bool
}

func NewEavesdropper() services.Service {
	return &Eavesdropper{
		cfg:       EavesdropperArgs{},
		basicAuth: authx.BasicAuth{},
		isStop:    false,
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
		s.basicAuth = authx.BasicAuth{}
		s.cfg = EavesdropperArgs{}
		s.log = nil
		s = nil
	}()
	s.isStop = true
}

func (s *Eavesdropper) Start(args interface{}, log *logrus.Logger) (err error) {
	s.log = log
	s.cfg = args.(EavesdropperArgs)

	err = s.InitBasicAuth()
	if err != nil {
		return
	}

	//s.basicAuth.Show()

	wl, err := utils.LoadWhiteList(*s.cfg.White)
	if err != nil {
		return err
	}

	var conds []goproxy.ReqCondition
	for _, v := range wl {
		conds = append(conds, goproxy.Not(goproxy.ReqHostMatches(regexp.MustCompile(fmt.Sprintf("^.*%s.*$", v)))))
	}

	for _, addr := range strings.Split(*s.cfg.Local, ",") {
		if addr != "" {

			proxy := goproxy.NewProxyHttpServer()

			// FIXME(mooofly): hardcode here right now
			proxy.Verbose = true
			proxy.Logger = s.log

			proxy.OnRequest(conds...).HandleConnect(goproxy.AlwaysReject)

			proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
				HandleConnect(goproxy.AlwaysMitm)

			proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
				HijackConnect(func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
					defer func() {
						if e := recover(); e != nil {
							ctx.Logf("error connecting to remote: %v", e)
							client.Write([]byte("HTTP/1.1 500 Cannot reach destination\r\n\r\n"))
						}
						client.Close()
					}()

					ua := req.Header.Get("User-Agent")
					s.log.Printf("User-Agent got: %s\n", ua)

					ua2 := ctx.Req.Header.Get("User-Agent")
					s.log.Printf("User-Agent2 got: %s\n", ua2)

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

			err = http.ListenAndServe(addr, proxy)
		}
	}
	return
}

func (s *Eavesdropper) Clean() {
	s.StopService()
}

func (s *Eavesdropper) InitBasicAuth() (err error) {
	s.basicAuth = authx.NewBasicAuth(s.log)

	if *s.cfg.AuthFile != "" {
		n, err := s.basicAuth.AddFromFile(*s.cfg.AuthFile)
		if err != nil {
			return err
		}
		s.log.Printf("auth data added from file %d, total:%d", n, s.basicAuth.Total())
	}

	if len(*s.cfg.Auth) > 0 {
		n := s.basicAuth.Add(*s.cfg.Auth)
		s.log.Printf("auth data added from CLI %d, total:%d", n, s.basicAuth.Total())
	}

	return
}

func (s *Eavesdropper) Verify(user, passwd string) bool {
	return s.basicAuth.Check(user, passwd)
}
