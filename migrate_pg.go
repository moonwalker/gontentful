package gontentful

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

const (
	newSchemaNameTpl = "%s_new"
	oldSchemaNameTpl = "%s_old"
	migrateSchemaTpl = `DROP SCHEMA IF EXISTS %[2]s CASCADE;
	CREATE SCHEMA IF NOT EXISTS %[1]s;
	ALTER SCHEMA %[1]s RENAME TO %[2]s;
	ALTER SCHEMA %[3]s RENAME TO %[1]s;`
	copyTableTpl = `INSERT INTO %[1]s.%[3]s SELECT * FROM %[2]s.%[3]s;`
)

func MigratePGSQL(databaseURL string, newSchemaName string, space *Space, types []*ContentType, cmaTypes []*ContentType, entries []*Entry, syncToken string, createFunctions bool) error {

	// 0) drop newSchema if exists
	drop := NewPGDrop(newSchemaName)
	err := drop.Exec(databaseURL)
	if err != nil {
		return err
	}

	// 1) re-create schema
	schema := NewPGSQLSchema(newSchemaName, space, cmaTypes, 0)
	err = schema.Exec(databaseURL)
	if err != nil {
		return err
	}

	// 2) sync data & save token
	sync := NewPGSyncSchema(newSchemaName, space, types, entries, true)
	err = sync.Exec(databaseURL)
	if err != nil {
		return err
	}
	err = SaveSyncToken(databaseURL, newSchemaName, syncToken)
	if err != nil {
		return err
	}

	// 3) create functions
	if createFunctions {
		funcs := NewPGFunctions(schema)
		err = funcs.Exec(databaseURL)
		if err != nil {
			return err
		}
	}

	return nil
}

func SwapSchemas(databaseURL string, schemaName string, oldSchemaName string, newSchemaName string) error {
	// rename (swap schemas)
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	txn, err := db.Beginx()
	if err != nil {
		return err
	}

	_, err = txn.Exec(fmt.Sprintf(migrateSchemaTpl, schemaName, oldSchemaName, newSchemaName))
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}

func CopyGameData(databaseURL string, schemaName string, newSchemaName string, tableNames []string) error {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	txn, err := db.Beginx()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for _, tn := range tableNames {
		_, err = txn.Exec(fmt.Sprintf(copyTableTpl, newSchemaName, schemaName, tn))
		if err != nil {
			return err
		}
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}
