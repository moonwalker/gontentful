package gontentful

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

type PGFunctions struct {
	Schema *PGSQLSchema
}

func NewPGFunctions(schema *PGSQLSchema) *PGFunctions {
	return &PGFunctions{
		Schema: schema,
	}
}

func (s *PGFunctions) Exec(databaseURL string) error {
	tmpl, err := template.New("").Parse(pgFuncTemplate)

	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s.Schema)
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
	if s.Schema.SchemaName != "" {
		// set schema in use
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.Schema.SchemaName))
		if err != nil {
			return err
		}
	}

	// ioutil.WriteFile("/tmp/func", buff.Bytes(), 0644)

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
