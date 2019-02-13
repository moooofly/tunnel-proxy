package basic

import (
	"fmt"
	logger "log"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	"github.com/moooofly/goproxy/services"
	"github.com/moooofly/goproxy/utils"
	"github.com/moooofly/goproxy/utils/authx"
)

const realm = "LLS-Realm"

type BasicArgs struct {
	Local    *string
	White    *string
	AuthFile *string
	Auth     *[]string
}

type Basic struct {
	cfg       BasicArgs
	basicAuth authx.BasicAuth
	log       *logger.Logger
	isStop    bool
}

func NewBasic() services.Service {
	return &Basic{
		cfg:       BasicArgs{},
		basicAuth: authx.BasicAuth{},
		isStop:    false,
	}
}

func (s *Basic) StopService() {
	defer func() {
		e := recover()
		if e != nil {
			s.log.Printf("Stop basic proxy crashed, %s", e)
		} else {
			s.log.Printf("service basic proxy stopped")
		}
		s.basicAuth = authx.BasicAuth{}
		s.cfg = BasicArgs{}
		s.log = nil
		s = nil
	}()
	s.isStop = true
}

func (s *Basic) Start(args interface{}, log *logger.Logger) (err error) {
	s.log = log
	s.cfg = args.(BasicArgs)

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

			// basic auth
			proxy.OnRequest().Do(auth.Basic(realm, s.Verify))
			proxy.OnRequest().HandleConnect(auth.BasicConnect(realm, s.Verify))

			err = http.ListenAndServe(addr, proxy)
			if err != nil {
				return
			}
		}
	}
	return
}

func (s *Basic) Clean() {
	s.StopService()
}

func (s *Basic) InitBasicAuth() (err error) {
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

func (s *Basic) Verify(user, passwd string) bool {
	return s.basicAuth.Check(user, passwd)

}
