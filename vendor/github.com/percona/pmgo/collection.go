package pmgo

import mgo "gopkg.in/mgo.v2"

// CollectionManager is an interface for mgo.Collection struct.
// All implemented methods returns interfaces when needed
type CollectionManager interface {
	Count() (int, error)
	Create(*mgo.CollectionInfo) error
	DropCollection() error
	//DropIndex(key ...string) error
	//DropIndexName(name string) error
	//EnsureIndex(index mgo.Index) error
	//EnsureIndexKey(key ...string) error
	Find(interface{}) QueryManager
	//FindId(id interface{}) *QueryManager
	//Indexes() (indexes []mgo.Index, err error)
	Insert(docs ...interface{}) error
	//NewIter(session *mgo.Session, firstBatch []bson.Raw, cursorId int64, err error) *mgo.Iter
	Pipe(interface{}) PipeManager
	//Remove(selector interface{}) error
	//RemoveAll(selector interface{}) (info *mgo.ChangeInfo, err error)
	//RemoveId(id interface{}) error
	//Repair() *mgo.Iter
	//Update(selector interface{}, update interface{}) error
	//UpdateAll(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
	//UpdateId(id interface{}, update interface{}) error
	//Upsert(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
	//UpsertId(id interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
	//With(s *mgo.Session) *CollectionManager
}

type Collection struct {
	collection *mgo.Collection
}

func NewCollectionManager(c *mgo.Collection) CollectionManager {
	return &Collection{
		collection: c,
	}
}

func (c *Collection) Count() (int, error) {
	return c.collection.Count()
}

func (c *Collection) Create(info *mgo.CollectionInfo) error {
	return c.collection.Create(info)
}

func (c *Collection) DropCollection() error {
	return c.collection.DropCollection()
}

func (c *Collection) Find(qu interface{}) QueryManager {
	return &Query{
		query: c.collection.Find(qu),
	}
}

func (c *Collection) Insert(docs ...interface{}) error {
	return c.collection.Insert(docs...)
}

func (c *Collection) Pipe(query interface{}) PipeManager {
	return &Pipe{
		pipe: c.collection.Pipe(query),
	}
}
