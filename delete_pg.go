package gontentful

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

const deleteTemplate = "DELETE FROM {{ .SchemaName }}.{{ .TableName }} WHERE _sys_id = '{{ .SysID }}';"

type PGDelete struct {
	SchemaName string
	TableName  string
	SysID      string
}

func NewPGDelete(schemaName string, sys *Sys) *PGDelete {
	tableName := ""
	if sys.Type == DELETED_ENTRY {
		tableName = toSnakeCase(sys.ContentType.Sys.ID)
	} else if sys.Type == DELETED_ASSET {
		tableName = ASSET_TABLE_NAME
	}
	return &PGDelete{
		SchemaName: schemaName,
		TableName:  tableName,
		SysID:      sys.ID,
	}
}

func (s *PGDelete) Exec(databaseURL string, schemaName string) error {
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

	if schemaName != "" {
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", schemaName))
		if err != nil {
			return err
		}
	}

	tmpl, err := template.New("").Parse(deleteTemplate)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	// fmt.Println(buff.String())

	_, err = txn.Exec(buff.String())
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return err
}
