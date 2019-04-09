package eavesdropper

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/moooofly/goproxy"
	"github.com/moooofly/goproxy/ext/auth"
	"github.com/moooofly/tunnel-proxy/services"
	"github.com/moooofly/tunnel-proxy/utils"
	"github.com/moooofly/tunnel-proxy/utils/authx"
	"github.com/moooofly/tunnel-proxy/utils/cipher"
	"github.com/sirupsen/logrus"
)

var key []byte = cipher.Base64Decode("a1vmCcIkAfAiu9u37YZ+SHX/JtRi4EP1yjRx6nZv0HY=")
var iv []byte = cipher.Base64Decode("P78Sw02O5m81WCbvEGRGjw==")

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

			// white list
			proxy.OnRequest(conds...).HandleConnect(goproxy.AlwaysReject)

			// header analysis
			proxy.OnRequest().HandleConnect(s.HeaderAnalysis())

			// basic auth

			// FIXME: comment out http auth for now
			//proxy.OnRequest().Do(auth.Basic(realm, s.Verify))
			proxy.OnRequest().HandleConnect(auth.BasicConnect(realm, s.Verify))

			/*
				proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
					HandleConnect(goproxy.AlwaysMitm)
			*/
			proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
				HijackConnect(func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
					// TODO(moooofly): do something here
				})

			err = http.ListenAndServe(addr, proxy)
			if err != nil {
				return
			}
		}
	}
	return
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-Ip")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = strings.Split(r.RemoteAddr, ":")[0]
		} else {
			ip = strings.Split(ip, ",")[0]
		}
	}
	return ip
}

// TODO: Log into file for later analysis
func (s *Eavesdropper) HeaderAnalysis() goproxy.HttpsHandler {
	return goproxy.FuncHttpsHandler(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		ua := ctx.Req.Header.Get("User-Agent")
		clientIP := getClientIP(ctx.Req)
		s.log.Printf("User-Agent: %s   remoteAddr: %s", ua, clientIP)

		// The format of User-Info is base64(Aes256(userId+login))
		ui := ctx.Req.Header.Get("User-Info")
		if ui == "" {
			//s.log.Println("Find no 'User-Info' header, Reject!")
			//return goproxy.RejectConnect, host
			s.log.Println("Find no 'User-Info' header, do nothing in this version!")
			return nil, host
		}

		IDs, err := cipher.AES_CBC_PKCS7_decode(cipher.Base64Decode(ui), key, iv)
		if err != nil {
			return goproxy.RejectConnect, host
		}
		s.log.Printf("userID,loginID: %s", string(IDs))

		return nil, host
	})
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
