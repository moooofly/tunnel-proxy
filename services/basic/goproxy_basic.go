package basic

import (
	logger "log"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	"github.com/moooofly/goproxy/services"
)

const realm = "LLS-Realm"

type BasicArgs struct {
	Local *string
	White *string
}

type Basic struct {
	cfg    BasicArgs
	log    *logger.Logger
	isStop bool
}

func NewBasic() services.Service {
	return &Basic{
		cfg:    BasicArgs{},
		isStop: false,
	}
}

func (s *Basic) StopService() {
	defer func() {
		e := recover()
		if e != nil {
			s.log.Printf("stop basic proxy crashed,%s", e)
		} else {
			s.log.Printf("service basic proxy stopped")
		}
		s.cfg = BasicArgs{}
		s.log = nil
		s = nil
	}()
	s.isStop = true
}

func (s *Basic) Start(args interface{}, log *logger.Logger) (err error) {
	s.log = log
	s.cfg = args.(BasicArgs)

	for _, addr := range strings.Split(*s.cfg.Local, ",") {
		if addr != "" {
			proxy := goproxy.NewProxyHttpServer()
			// FIXME(mooofly): hardcode here right now
			proxy.Verbose = true
			proxy.Logger = s.log

			proxy.OnRequest().Do(auth.Basic(realm, verify))
			proxy.OnRequest().HandleConnect(auth.BasicConnect(realm, verify))

			// white list
			// FIXME(mooofly): hardcode here right now
			proxy.OnRequest(
				goproxy.Not(goproxy.ReqHostMatches(regexp.MustCompile("^.*llsapp.com.*$"))),
				goproxy.Not(goproxy.ReqHostMatches(regexp.MustCompile("^.*liulishuo.com.*$"))),
				goproxy.Not(goproxy.ReqHostMatches(regexp.MustCompile("^.*llscdn.com.*$"))),
			).HandleConnect(goproxy.AlwaysReject)

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

// FIXME(moooofly): should be more general
func verify(user, passwd string) bool {
	return user == "foo" && passwd == "bar"
}
