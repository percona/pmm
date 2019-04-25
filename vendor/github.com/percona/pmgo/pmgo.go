package pmgo

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	mgo "gopkg.in/mgo.v2"
)

type Dialer interface {
	Dial(string) (SessionManager, error)
	DialWithInfo(*DialInfo) (SessionManager, error)
	DialWithTimeout(string, time.Duration) (SessionManager, error)
}

type DialInfo struct {
	SSLPEMKeyFile string
	SSLCAFile     string

	Addrs          []string
	Direct         bool
	Timeout        time.Duration
	FailFast       bool
	Database       string
	ReplicaSetName string
	Source         string
	Service        string
	ServiceHost    string
	Mechanism      string
	Username       string
	Password       string
	PoolLimit      int
	DialServer     func(addr *mgo.ServerAddr) (net.Conn, error)
}

type dialer struct{}

func NewDialInfo(src *mgo.DialInfo) *DialInfo {
	return &DialInfo{
		Addrs:          src.Addrs,
		Direct:         src.Direct,
		Timeout:        src.Timeout,
		FailFast:       src.FailFast,
		Database:       src.Database,
		ReplicaSetName: src.ReplicaSetName,
		Source:         src.Source,
		Service:        src.Service,
		ServiceHost:    src.ServiceHost,
		Mechanism:      src.Mechanism,
		Username:       src.Username,
		Password:       src.Password,
		PoolLimit:      src.PoolLimit,
		DialServer:     src.DialServer,
	}
}

func ParseURL(url string) (*DialInfo, error) {
	di, err := mgo.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return NewDialInfo(di), nil
}

func NewDialer() Dialer {
	return new(dialer)
}

func (d *dialer) Dial(url string) (SessionManager, error) {
	s, err := mgo.Dial(url)
	se := &Session{
		session: s,
	}
	return se, err
}

func (d *dialer) DialWithInfo(info *DialInfo) (SessionManager, error) {
	mgoInfo := &mgo.DialInfo{
		Addrs:          info.Addrs,
		Direct:         info.Direct,
		Timeout:        info.Timeout,
		FailFast:       info.FailFast,
		Database:       info.Database,
		ReplicaSetName: info.ReplicaSetName,
		Source:         info.Source,
		Service:        info.Service,
		ServiceHost:    info.ServiceHost,
		Mechanism:      info.Mechanism,
		Username:       info.Username,
		Password:       info.Password,
		PoolLimit:      info.PoolLimit,
		DialServer:     info.DialServer,
	}

	if info.SSLCAFile != "" || info.SSLPEMKeyFile != "" {
		tlsConfig := &tls.Config{}

		if info.SSLCAFile != "" {
			if _, err := os.Stat(info.SSLCAFile); os.IsNotExist(err) {
				return nil, err
			}

			roots := x509.NewCertPool()
			var ca []byte
			var err error

			if ca, err = ioutil.ReadFile(info.SSLCAFile); err != nil {
				return nil, fmt.Errorf("invalid pem file: %s", err.Error())
			}
			roots.AppendCertsFromPEM(ca)
			tlsConfig.RootCAs = roots

		}

		if info.SSLPEMKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(info.SSLPEMKeyFile, info.SSLPEMKeyFile)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		mgoInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}

		mgoInfo.Source = "$external"
		mgoInfo.Mechanism = "MONGODB-X509"
	}

	s, err := mgo.DialWithInfo(mgoInfo)

	se := &Session{
		session: s,
	}
	return se, err
}

func (d *dialer) DialWithTimeout(url string, timeout time.Duration) (SessionManager, error) {
	s, err := mgo.DialWithTimeout(url, timeout)
	se := &Session{
		session: s,
	}
	return se, err
}
