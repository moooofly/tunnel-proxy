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
			proxy.OnRequest().Do(auth.Basic(realm, verify))
			proxy.OnRequest().HandleConnect(auth.BasicConnect(realm, verify))

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
