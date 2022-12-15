package client

//go:generate ../../../../../bin/ifacemaker -f client.go -s Client -i kubeClientConnector -p client -o interface.go
//go:generate ../../../../../bin/mockery -name=kubeClientConnector -case=snake -inpkg -testonly
