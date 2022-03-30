package gontentful

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
)

var overwritableFields = map[string]bool{
	"name":        true,
	"content":     true,
	"description": true,
	"priority":    true,
}

var funcMap = template.FuncMap{
	"ToLower": strings.ToLower,
	"Overwritable": func(f string) bool {
		return overwritableFields[f]
	},
}

type PGFunctions struct {
	Schema *PGSQLSchema
}

func NewPGFunctions(schema *PGSQLSchema) *PGFunctions {
	return &PGFunctions{
		Schema: schema,
	}
}

func (s *PGFunctions) Exec(databaseURL string) error {
	tmpl, err := template.New("").Funcs(funcMap).Parse(pgFuncTemplate)

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

func (s *PGFunctions) Render() (string, error) {
	tmpl, err := template.New("").Funcs(funcMap).Parse(pgFuncTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s.Schema)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}
