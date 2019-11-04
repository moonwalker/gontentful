package gontentful

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

type PGFunctions struct {
	SchemaName string
}

func NewPGFunctions(schemaName string) *PGFunctions {
	return &PGFunctions{
		SchemaName: schemaName,
	}
}

func (s *PGFunctions) Exec(databaseURL string) error {
	tmpl, err := template.New("").Parse(pgFuncTemplate)

	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	db, err := sqlx.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	txn, err := db.Beginx()
	if err != nil {
		return err
	}
	if s.SchemaName != "" {
		// set schema in use
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.SchemaName))
		if err != nil {
			return err
		}
	}
	_, err = txn.Exec(buff.String())
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}
	return nil
}
