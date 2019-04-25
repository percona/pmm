package pmgo

import (
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type SessionManager interface {
	BuildInfo() (info mgo.BuildInfo, err error)
	Clone() SessionManager
	Close()
	Copy() SessionManager
	DB(name string) DatabaseManager
	DatabaseNames() (names []string, err error)
	EnsureSafe(safe *mgo.Safe)
	FindRef(ref *mgo.DBRef) QueryManager
	Fsync(async bool) error
	FsyncLock() error
	FsyncUnlock() error
	LiveServers() (addrs []string)
	Login(cred *mgo.Credential) error
	LogoutAll()
	Mode() mgo.Mode
	New() SessionManager
	Ping() error
	Refresh()
	ResetIndexCache()
	Run(cmd interface{}, result interface{}) error
	Safe() (safe *mgo.Safe)
	SelectServers(tags ...bson.D)
	SetBatch(n int)
	SetBypassValidation(bypass bool)
	SetCursorTimeout(d time.Duration)
	SetMode(consistency mgo.Mode, refresh bool)
	SetPoolLimit(limit int)
	SetPrefetch(p float64)
	SetSafe(safe *mgo.Safe)
	SetSocketTimeout(d time.Duration)
	SetSyncTimeout(d time.Duration)
}

type Session struct {
	session *mgo.Session
}

// This methos allows to use mgo's dbtest.DBServer in pmgo tests.
// Example:
// var Server dbtest.DBServer
// tempDir, _ := ioutil.TempDir("", "testing")
// Server.SetPath(tempDir)
// session := NewSessionManager(Server.Session())
func NewSessionManager(s *mgo.Session) SessionManager {
	return &Session{
		session: s,
	}
}

func (s *Session) BuildInfo() (info mgo.BuildInfo, err error) {
	return s.session.BuildInfo()
}

func (s *Session) Close() {
	s.session.Close()
}

func (s *Session) Clone() SessionManager {
	return &Session{
		session: s.session.Clone(),
	}
}

func (s *Session) Copy() SessionManager {
	return &Session{
		session: s.session.Copy(),
	}
}

func (s *Session) DB(name string) DatabaseManager {
	d := &Database{
		db: s.session.DB(name),
	}
	return d
}

func (s *Session) DatabaseNames() (names []string, err error) {
	return s.session.DatabaseNames()
}

func (s *Session) EnsureSafe(safe *mgo.Safe) {
	s.session.EnsureSafe(safe)
}

func (s *Session) FindRef(ref *mgo.DBRef) QueryManager {
	return &Query{s.session.FindRef(ref)}
}

func (s *Session) Fsync(async bool) error {
	return s.session.Fsync(async)
}

func (s *Session) FsyncLock() error {
	return s.session.FsyncLock()
}

func (s *Session) FsyncUnlock() error {
	return s.session.FsyncUnlock()
}

func (s *Session) LiveServers() (addrs []string) {
	return s.session.LiveServers()
}

func (s *Session) Login(cred *mgo.Credential) error {
	return s.session.Login(cred)
}

func (s *Session) LogoutAll() {
	s.session.LogoutAll()
}

func (s *Session) Mode() mgo.Mode {
	return s.session.Mode()
}

func (s *Session) New() SessionManager {
	return &Session{s.session.New()}
}

func (s *Session) Run(cmd interface{}, result interface{}) error {
	return s.session.Run(cmd, result)
}

func (s *Session) Safe() (safe *mgo.Safe) {
	return s.session.Safe()
}

func (s *Session) SelectServers(tags ...bson.D) {
	s.session.SelectServers(tags...)
}

func (s *Session) SetBatch(n int) {
	s.session.SetBatch(n)
}

func (s *Session) SetBypassValidation(bypass bool) {
	s.session.SetBypassValidation(bypass)
}

func (s *Session) SetCursorTimeout(d time.Duration) {
	s.session.SetCursorTimeout(d)
}

func (s *Session) Ping() error {
	return s.session.Ping()
}

func (s *Session) Refresh() {
	s.session.Refresh()
}

func (s *Session) ResetIndexCache() {
	s.session.ResetIndexCache()
}

func (s *Session) SetMode(consistency mgo.Mode, refresh bool) {
	s.session.SetMode(consistency, refresh)
}
func (s *Session) SetPoolLimit(limit int) {
	s.session.SetPoolLimit(limit)
}

func (s *Session) SetPrefetch(p float64) {
	s.session.SetPrefetch(p)
}

func (s *Session) SetSafe(safe *mgo.Safe) {
	s.session.SetSafe(safe)
}

func (s *Session) SetSocketTimeout(d time.Duration) {
	s.session.SetSocketTimeout(d)
}
func (s *Session) SetSyncTimeout(d time.Duration) {
	s.session.SetSyncTimeout(d)
}
