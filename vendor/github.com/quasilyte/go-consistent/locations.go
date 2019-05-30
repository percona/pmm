package main

import (
	"go/token"
)

// TODO(Quasilyte): optimized implementation.
type locationMap struct {
	m    map[token.Position]int
	keys []token.Position
}

func newLocationMap() *locationMap {
	return &locationMap{m: make(map[token.Position]int)}
}

func (locs *locationMap) Insert(filename string, line, column int) int {
	key := token.Position{Filename: filename, Line: line, Column: column}
	if id, ok := locs.m[key]; ok {
		return id
	}
	id := len(locs.m)
	locs.keys = append(locs.keys, key)
	locs.m[key] = id
	return id
}

func (locs *locationMap) Get(id int) token.Position {
	return locs.keys[id]
}
