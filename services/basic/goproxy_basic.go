package basic

import (
	logger "log"
	"net/http"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/moooofly/goproxy/services"
)

type BasicArgs struct {
	Local *string
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
			//proxy.Verbose = *verbose
			err = http.ListenAndServe(addr, proxy)
		}
	}
	return
}

func (s *Basic) Clean() {
	s.StopService()
}
