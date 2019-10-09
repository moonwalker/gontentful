package gontentful

import (
	"bytes"
	"database/sql"
	"text/template"
)

const dropTemplate = "DROP SCHEMA IF EXISTS {{ $.SchemaName }} CASCADE;"

type PGDrop struct {
	SchemaName string
}

func NewPGDrop(schemaName string) *PGDrop {

	return &PGDrop{
		SchemaName: schemaName,
	}
}

func (s *PGDrop) Exec(databaseURL string) error {
	db, _ := sql.Open("postgres", databaseURL)

	defer db.Close()

	tmpl, err := template.New("").Parse(dropTemplate)

	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	// fmt.Println(buff.String())

	_, err = db.Exec(buff.String())

	return err
}
