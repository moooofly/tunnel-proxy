package utils

import (
	"io/ioutil"
	"os"
	"strings"
)

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func LoadWhiteList(f string) (wl []string, err error) {
	if pathExists(f) {
		ct, err := ioutil.ReadFile(f)
		if err != nil {
			return wl, err
		}
		for _, line := range strings.Split(string(ct), "\n") {
			line = strings.Trim(line, "\r \t")
			if line != "" {
				wl = append(wl, line)
			}
		}
	}
	return
}
