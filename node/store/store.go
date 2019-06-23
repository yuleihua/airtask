// Copyright 2018 The huayulei_2003@hotmail.com Authors
// This file is part of the airfk library.
//
// The airfk library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The airfk library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the airfk library. If not, see <http://www.gnu.org/licenses/>.
package store

import (
	"airman.com/airfk/pkg/leveldb"
)

// Store is a wrapper as table.
type Store struct {
	db     *leveldb.LevelDB
	prefix []byte
}

// NewStore returns a database object.
func NewStore(db *leveldb.LevelDB, prefix []byte) *Store {
	return &Store{
		db:     db,
		prefix: prefix,
	}
}

// Close ...
func (t *Store) Close() error {
	t.db.Close()
	return nil
}

// Has retrieves if a prefixed key.
func (t *Store) Has(key []byte) (bool, error) {
	return t.db.Has(append(t.prefix, key...))
}

// Get retrieves the given prefixed key.
func (t *Store) Get(key []byte) ([]byte, error) {
	return t.db.Get(append(t.prefix, key...))
}

// Put inserts the given value into the database.
func (t *Store) Put(key []byte, value []byte) error {
	return t.db.Put(append(t.prefix, key...), value)
}

// Delete removes the given prefixed key from the database.
func (t *Store) Delete(key []byte) error {
	return t.db.Delete(append(t.prefix, key...))
}
