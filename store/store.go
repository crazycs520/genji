// Package store consists mainly of the Store type, which is a high level representation of a table meant to be used by code generated by Genji.
// It is to be used to simplify common operations such as fetching one record, listing them or reindexing a table.
package store

import (
	"errors"
	"fmt"

	"github.com/asdine/genji"
	"github.com/asdine/genji/engine"
	"github.com/asdine/genji/record"
	"github.com/asdine/genji/table"
)

// Store is a high level representation of a table.
// It provides helpers to manage the underlying table.
// It can be used used within or out of a transaction, automatically opening one when needed.
type Store struct {
	db        *genji.DB
	tx        *genji.Tx
	tableName string
	schema    *record.Schema
	indexes   []string
}

// New creates a store for the specified table. If schema is non nil, the Store will
// manage the table as a schemaful table. If schema is nil, the table will be considered as
// schemaless.
// New returns a long lived store that automatically creates its own transactions when needed.
func New(db *genji.DB, tableName string, schema *record.Schema, indexes []string) *Store {
	return &Store{
		db:        db,
		tableName: tableName,
		schema:    schema,
		indexes:   indexes,
	}
}

// NewWithTx creates a store valid for the lifetime of the given transaction.
func NewWithTx(tx *genji.Tx, tableName string, schema *record.Schema, indexes []string) *Store {
	return &Store{
		tx:        tx,
		tableName: tableName,
		schema:    schema,
		indexes:   indexes,
	}
}

func (s *Store) run(writable bool, fn func(tx *genji.Tx) error) error {
	tx := s.tx
	var err error

	if tx == nil {
		tx, err = s.db.Begin(writable)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	err = fn(tx)
	if err != nil {
		return err
	}

	if s.tx == nil && writable {
		return tx.Commit()
	}

	return nil
}

// View starts a read only transaction, runs fn and automatically rolls it back.
// If the store has been created within an existing transaction, View
// will reuse it instead of creating one.
func (s *Store) View(fn func(tx *genji.Tx) error) error {
	return s.run(false, fn)
}

// Update starts a read-write transaction, runs fn and automatically commits it.
// If the store has been created within an existing transaction, Update
// will reuse it instead of creating one.
// If fn returns an error, the transaction is rolled back, unless the store has
// been created with NewWithTx.
func (s *Store) Update(fn func(tx *genji.Tx) error) error {
	return s.run(true, fn)
}

// ViewTable starts a read only transaction, fetches the underlying table, calls fn with that table
// and automatically rolls back the transaction.
// If the store has been created within an existing transaction, ViewTable
// will reuse it instead of creating one.
func (s *Store) ViewTable(fn func(*genji.Table) error) error {
	return s.View(func(tx *genji.Tx) error {
		tb, err := tx.Table(s.tableName)
		if err != nil {
			return err
		}

		return fn(tb)
	})
}

// UpdateTable starts a read/write transaction, fetches the underlying table, calls fn with that table
// and automatically commits the transaction.
// If the store has been created within an existing transaction, UpdateTable
// will reuse it instead of creating one.
// If fn returns an error, the transaction is rolled back, unless the store has
// been created with NewWithTx.
func (s *Store) UpdateTable(fn func(*genji.Table) error) error {
	return s.Update(func(tx *genji.Tx) error {
		tb, err := tx.Table(s.tableName)
		if err != nil {
			return err
		}

		return fn(tb)
	})
}

// Init makes sure the table exists. No error is returned if the table already exists.
// If the store was created using a schema, checks if the given schema matches the one stored in the table.
func (s *Store) Init() error {
	return s.Update(func(tx *genji.Tx) error {
		var err error
		if s.schema != nil {
			err = tx.CreateTableWithSchema(s.tableName, s.schema)
		} else {
			err = tx.CreateTable(s.tableName)
		}

		if err != nil && err != engine.ErrTableAlreadyExists {
			return err
		}

		tb, err := tx.Table(s.tableName)
		if err != nil {
			return err
		}

		schema, schemaful := tb.Schema()
		if s.schema != nil {
			if !schemaful {
				return errors.New("the table is schemaless, yet a schema has been passed")
			}

			if !s.schema.Equal(&schema) {
				return fmt.Errorf("given schema doesn't match current one: expected %q got %q", schema, s.schema)
			}
		} else {
			if schemaful {
				return errors.New("the table is schemaful, yet no schema has been passed")
			}
		}

		if s.indexes != nil {
			for _, fname := range s.indexes {
				err = tx.CreateIndex(s.tableName, fname)
				if err != nil && err != engine.ErrIndexAlreadyExists {
					return err
				}
			}
		}

		return nil
	})
}

// Insert a record in the table and return the recordID.
func (s *Store) Insert(r record.Record) (recordID []byte, err error) {
	err = s.UpdateTable(func(t *genji.Table) error {
		recordID, err = t.Insert(r)
		return err
	})
	return
}

// Get a record by recordID.
// If the recordID doesn't exist, returns table.ErrRecordNotFound.
func (s *Store) Get(recordID []byte) (rec record.Record, err error) {
	err = s.ViewTable(func(t *genji.Table) error {
		rec, err = t.Record(recordID)
		return err
	})
	return
}

// Delete a record by recordID.
// If the recordID doesn't exist, returns table.ErrRecordNotFound.
func (s *Store) Delete(recordID []byte) error {
	return s.UpdateTable(func(t *genji.Table) error {
		return t.Delete(recordID)
	})
}

// Drop the table.
func (s *Store) Drop() error {
	return s.Update(func(tx *genji.Tx) error {
		return tx.DropTable(s.tableName)
	})
}

// DropIndex removes an index from the table.
func (s *Store) DropIndex(fieldName string) error {
	return s.Update(func(tx *genji.Tx) error {
		return tx.DropIndex(s.tableName, fieldName)
	})
}

// List records from the specified offset. If the limit is equal to -1, it returns all records after the selected offset.
func (s *Store) List(offset, limit int, fn func(recordID []byte, r record.Record) error) error {
	return s.ViewTable(func(t *genji.Table) error {
		return table.NewBrowser(t).Offset(offset).Limit(limit).ForEach(fn).Err()
	})
}

// Replace a record by another one.
func (s *Store) Replace(recordID []byte, r record.Record) error {
	return s.UpdateTable(func(t *genji.Table) error {
		return t.Replace(recordID, r)
	})
}

// Truncate the table.
func (s *Store) Truncate() error {
	return s.UpdateTable(func(t *genji.Table) error {
		return t.Truncate()
	})
}