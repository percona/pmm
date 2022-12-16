package client

//go:generate ../../../../../bin/ifacemaker -f client.go -s Client -i KubeClientConnector -p client -o interface.go
//go:generate ../../../../../bin/mockery -name=KubeClientConnector -case=snake -inpkg
