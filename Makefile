USER := foo
PASS := bar

all:
	@echo "Usage:"
	@echo " 1. make (show Usage)"
	@echo " 2. make test (both succ and fail)"
	@echo " 3. make succ"
	@echo " 4. make fail"

test: succ fail

succ:
	@echo "test with white-list"
	@echo
	@echo "-----"
	curl -Lv --proxy http://${USER}:${PASS}@127.0.0.1:8080 https://apineo.llsapp.com
	@echo
	@echo "-----"
	curl -Lv --proxy http://${USER}:${PASS}@127.0.0.1:8080 https://cdn.llscdn.com/ping
	@echo
	@echo "-----"
	curl -Lv --proxy http://${USER}:${PASS}@127.0.0.1:8080 https://www.liulishuo.com
	@echo

fail:
	@echo "test with httpbin.org and baidu.com"
	@echo
	@echo "-----"
	-curl -Lv --proxy http://${USER}:${PASS}@127.0.0.1:8080 https://httpbin.org/ip
	@echo
	@echo "-----"
	-curl -Lv --proxy http://${USER}:${PASS}@127.0.0.1:8080 https://baidu.com
	@echo

