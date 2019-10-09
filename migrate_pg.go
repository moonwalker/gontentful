package gontentful

import (
	"database/sql"
	"fmt"
)

const (
	newSchemaNameTpl = "%s_new"
	oldSchemaNameTpl = "%s_old"
	migrateSchemaTpl = `ALTER SCHEMA %[1]s RENAME TO %[2]s;
	ALTER SCHEMA %[3]s RENAME TO %[1]s;
	DROP SCHEMA %[2]s CASCADE;`
)

func MigratePGSQL(databaseURL string, schemaName string,
	space *Space, types []*ContentType, cmaTypes []*ContentType, entries []*Entry, syncToken string) error {

	newSchemaName := fmt.Sprintf(newSchemaNameTpl, schemaName)
	oldSchemaName := fmt.Sprintf(oldSchemaNameTpl, schemaName)

	// 1) re-create schema
	schema := NewPGSQLSchema(newSchemaName, space, cmaTypes)
	err := schema.Exec(databaseURL)
	if err != nil {
		return err
	}

	// 2) sync data & save token
	sync := NewPGSyncSchema(newSchemaName, types, entries, true)
	err = sync.Exec(databaseURL)
	if err != nil {
		return err
	}
	err = SaveSyncToken(databaseURL, newSchemaName, syncToken)
	if err != nil {
		return err
	}

	// 3) rename (swap schemas)
	db, _ := sql.Open("postgres", databaseURL)
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(migrateSchemaTpl, schemaName, oldSchemaName, newSchemaName))
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}
