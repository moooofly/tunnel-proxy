package authx

import (
	"io/ioutil"
	"strings"

	"github.com/moooofly/goproxy/utils/mapx"
	"github.com/sirupsen/logrus"
)

type BasicAuth struct {
	data mapx.ConcurrentMap
	log  *logrus.Logger
}

func NewBasicAuth(log *logrus.Logger) BasicAuth {
	return BasicAuth{
		data: mapx.NewConcurrentMap(),
		log:  log,
	}
}

func (ba *BasicAuth) AddFromFile(file string) (n int, err error) {
	ct, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	pairs := strings.Split(strings.Replace(string(ct), "\r", "", -1), "\n")
	for _, pair := range pairs {
		if strings.HasPrefix(pair, "#") {
			continue
		}
		u := strings.Split(strings.Trim(pair, " "), ":")
		if len(u) == 2 {
			ba.data.Set(u[0], u[1])
			n++
		}
	}
	return
}

func (ba *BasicAuth) Add(pairs []string) (n int) {
	for _, pair := range pairs {
		u := strings.Split(pair, ":")
		if len(u) == 2 {
			ba.data.Set(u[0], u[1])
			n++
		}
	}
	return
}
func (ba *BasicAuth) Delete(pair []string) {
	for _, u := range pair {
		ba.data.Remove(u)
	}
}

func (ba *BasicAuth) Total() (n int) {
	n = ba.data.Count()
	return
}

func (ba *BasicAuth) Check(user, pass string) (ok bool) {
	if got, ok := ba.data.Get(user); ok {
		return got.(string) == pass
	}
	return false
}

func (ba *BasicAuth) Show() {
	for i, key := range ba.data.Keys() {
		value, _ := ba.data.Get(key)
		ba.log.Printf("[%d] %s:%s\n", i, key, value)
	}
}
