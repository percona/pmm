// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cache contains generic cache implementation.
package cache

import (
	"container/list"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ErrWrongType is returned by different functions when the type of the passed argument cannot be used to create or manipulate Cache.
var ErrWrongType = errors.New("wrong argument type")

// Cache provides cached access to performance statistics tables.
// It retains data longer than those tables.
// Intended to store various subtypes of map.
type Cache struct {
	typ       reflect.Type
	retain    time.Duration
	sizeLimit uint
	l         *logrus.Entry

	rw        sync.RWMutex
	items     map[interface{}]*list.Element
	itemsList *list.List
	updatedN  uint
	addedN    uint
	removedN  uint
	trimmedN  uint
}

// cacheItem is an element stored in Cache.
type cacheItem struct {
	key   interface{}
	value interface{}
	added time.Time
}

// New creates new Cache.
// Argument typ is an instance of type to be stored in Cache, must be a map with chosen key and value types.
func New(typ interface{}, retain time.Duration, sizeLimit uint, l *logrus.Entry) (*Cache, error) {
	if reflect.TypeOf(typ).Kind() != reflect.Map {
		return nil, fmt.Errorf("%w: typ must be of map kind", ErrWrongType)
	}
	return &Cache{
		typ:       reflect.TypeOf(typ),
		retain:    retain,
		sizeLimit: sizeLimit,
		l:         l,
		items:     make(map[interface{}]*list.Element),
		itemsList: list.New(),
	}, nil
}

// Get fills dest argument with all current items if the cache.
func (c *Cache) Get(dest interface{}) error {
	if reflect.TypeOf(dest) != c.typ {
		return fmt.Errorf("%w: must be %v, got %v", ErrWrongType, c.typ, reflect.TypeOf(dest))
	}
	c.rw.RLock()
	defer c.rw.RUnlock()

	m := reflect.ValueOf(dest)
	for k, v := range c.items {
		m.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v.Value.(*cacheItem).value)) //nolint:forcetypeassert
	}
	return nil
}

// Set removes expired items from cache, then adds current items, then trims the cache if it's length is more than specified.
func (c *Cache) Set(current interface{}) error {
	if reflect.TypeOf(current) != c.typ {
		return fmt.Errorf("%w: must be %v, got %v", ErrWrongType, c.typ, reflect.TypeOf(current))
	}

	c.rw.Lock()
	defer c.rw.Unlock()
	now := time.Now()
	var wasTrimmed bool

	var next *list.Element
	for e := c.itemsList.Front(); e != nil && now.Sub(e.Value.(*cacheItem).added) > c.retain; e = next { //nolint:forcetypeassert
		c.removedN++
		next = e.Next()
		delete(c.items, c.itemsList.Remove(e).(*cacheItem).key) //nolint:forcetypeassert
	}

	m := reflect.ValueOf(current)
	iter := m.MapRange()
	for iter.Next() {
		key := iter.Key().Interface()
		value := iter.Value().Interface()
		if e, ok := c.items[key]; ok {
			c.updatedN++
			e.Value.(*cacheItem).added = now   //nolint:forcetypeassert
			e.Value.(*cacheItem).value = value //nolint:forcetypeassert
			c.itemsList.MoveToBack(e)
		} else {
			c.addedN++
			c.items[key] = c.itemsList.PushBack(&cacheItem{key, value, now})

			if uint(len(c.items)) > c.sizeLimit {
				delete(c.items, c.itemsList.Remove(c.itemsList.Front()).(*cacheItem).key) //nolint:forcetypeassert
				c.removedN++
				c.trimmedN++
				wasTrimmed = true
			}
		}
	}
	if wasTrimmed {
		c.l.Debugf("Cache size exceeded the limit of %d items and the oldest values were trimmed. "+
			"Now the oldest query in the cache is of time %s",
			c.sizeLimit, c.itemsList.Front().Value.(*cacheItem).added.UTC().Format("2006-01-02T15:04:05Z")) //nolint:forcetypeassert
	}
	return nil
}

// Stats returns Cache statistics.
func (c *Cache) Stats() Stats {
	c.rw.RLock()
	defer c.rw.RUnlock()

	oldest := time.Unix(0, 0)
	newest := time.Unix(0, 0)
	if len(c.items) != 0 {
		oldest = c.itemsList.Front().Value.(*cacheItem).added //nolint:forcetypeassert
		newest = c.itemsList.Back().Value.(*cacheItem).added  //nolint:forcetypeassert
	}

	return Stats{
		Current:  uint(len(c.items)),
		UpdatedN: c.updatedN,
		AddedN:   c.addedN,
		RemovedN: c.removedN,
		TrimmedN: c.trimmedN,
		Oldest:   oldest,
		Newest:   newest,
	}
}

// Len returns the current number of elements in Cache.
func (c *Cache) Len() int {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return len(c.items)
}

// Capacity returns the maximum number of elements in Cache.
func (c *Cache) Capacity() uint {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.sizeLimit
}

// Stats contains Cache statistics.
type Stats struct {
	Current  uint
	UpdatedN uint
	AddedN   uint
	RemovedN uint
	TrimmedN uint
	Oldest   time.Time
	Newest   time.Time
}

func (s Stats) String() string {
	d := s.Newest.Sub(s.Oldest)
	return fmt.Sprintf("current=%d: updated=%d added=%d removed=%d; %s - %s (%s)",
		s.Current, s.UpdatedN, s.AddedN, s.RemovedN,
		s.Oldest.UTC().Format("2006-01-02T15:04:05Z"), s.Newest.UTC().Format("2006-01-02T15:04:05Z"), d)
}
