package gontentful

import (
	"bytes"
	"database/sql"
	"fmt"
	"text/template"
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

	db, _ := sql.Open("postgres", databaseURL)
	defer db.Close()

	txn, err := db.Begin()
	if err != nil {
		return err
	}

	// set schema in use
	_, err = db.Exec(fmt.Sprintf("SET search_path='%s'", s.SchemaName))
	if err != nil {
		return err
	}

	_, err = db.Exec(buff.String())
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}
	return nil
}
