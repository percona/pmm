package pmgo

import "gopkg.in/mgo.v2/dbtest"

type DBTestServer interface {
	Session() SessionManager
	SetPath(dbpath string)
	Stop()
	Wipe()
}

type DBTServer struct {
	dbserver dbtest.DBServer
}

func NewDBServer() DBTestServer {
	return &DBTServer{}
}

func (d *DBTServer) Session() SessionManager {
	se := &Session{
		session: d.dbserver.Session(),
	}
	return se
}

func (d *DBTServer) SetPath(dbpath string) {
	d.dbserver.SetPath(dbpath)
}

func (d *DBTServer) Stop() {
	d.dbserver.Stop()
}

func (d *DBTServer) Wipe() {
	d.dbserver.Wipe()
}
