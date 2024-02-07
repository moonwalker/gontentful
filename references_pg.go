package gontentful

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

type PGReferences struct {
	Schema *PGSQLSchema
}

func NewPGReferences(schema *PGSQLSchema) *PGReferences {
	return &PGReferences{
		Schema: schema,
	}
}

func (s *PGReferences) Exec(databaseURL string) error {
	str, err := s.Render()
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
	defer txn.Rollback()

	if s.Schema.SchemaName != "" {
		// set schema in use
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.Schema.SchemaName))
		if err != nil {
			return err
		}
	}

	_, err = txn.Exec(str)
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *PGReferences) Render() (string, error) {
	tmpl, err := template.New("referencesTemplate").Parse(pgReferencesTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return "", err
	}

	// os.WriteFile("/tmp/refs", buff.Bytes(), 0644)

	return buff.String(), nil
}
