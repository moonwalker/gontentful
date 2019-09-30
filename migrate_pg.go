package gontentful

import (
	"database/sql"
	"fmt"
)

const (
	tempSchemaNameTpl = "%s_NEW"
	migrateSchemaTpl  = `ALTER SCHEMA %[1]s RENAME TO %[1]s_OLD;
	ALTER SCHEMA %[1]s_NEW RENAME TO %[1]s;
	DROP SCHEMA %[1]s_OLD CASCADE;`
)

func MigratePGSQL(databaseURL string, schemaName string,
	space *Space, types []*ContentType, cmaTypes []*ContentType, entries []*Entry) error {

	tmpSchemaName := fmt.Sprintf(tempSchemaNameTpl, schemaName)

	// 1) schema
	schema := NewPGSQLSchema(tmpSchemaName, true, space, cmaTypes)
	err := schema.Exec(databaseURL)
	if err != nil {
		return err
	}

	// 2) data
	sync := NewPGSyncSchema(tmpSchemaName, types, entries, true)
	err = sync.Exec(databaseURL)
	if err != nil {
		return err
	}

	// 3) rename
	db, _ := sql.Open("postgres", databaseURL)
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(migrateSchemaTpl, schemaName))
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}
