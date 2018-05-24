all: systemd upstart supervisord sysv clean

test:
	@go test -o service.test -c ..

clean:
	-rm service.test

sysv: test
	@echo sysv
	@cp service.test sysv/
	@docker build -q --tag="service.test.sysv" sysv
	@-docker run service.test.sysv
	@-docker rm $(shell docker ps -l -q)
	@-docker rmi -f service.test.sysv
	@-rm sysv/service.test

systemd: test
	@echo systemd
	@cp service.test systemd/
	@docker build -q --tag="service.test.systemd" systemd
	@-docker run --privileged -v /sys/fs/cgroup:/sys/fs/cgroup:ro service.test.systemd
	@-docker rm $(shell docker ps -l -q)
	@-docker rmi -f service.test.systemd
	@-rm systemd/service.test

upstart: test
	@echo upstart
	@cp service.test upstart/
	@docker build -q --tag="service.test.upstart" upstart
	@-docker run service.test.upstart
	@-docker rm $(shell docker ps -l -q)
	@-docker rmi -f service.test.upstart
	@-rm upstart/service.test

supervisord: test
	@echo supervisord
	@cp service.test supervisord/
	@docker build -q --tag="service.test.supervisord" supervisord
	@-docker run service.test.supervisord
	@-docker rm $(shell docker ps -l -q)
	@-docker rmi -f service.test.supervisord
	@-rm supervisord/service.test
