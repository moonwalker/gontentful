package operations

import (
	"bytes"
	"text/template"

	"github.com/jmoiron/sqlx"

	"github.com/moonwalker/gontentful/schema"
)

type PGGames struct {
	SchemaName string
}

func NewPGGames(schemaName string) *PGGames {

	return &PGGames{
		SchemaName: schemaName,
	}
}

func (s *PGGames) Exec(databaseURL string) error {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return err
	}

	defer db.Close()

	tmpl, err := template.New("").Parse(schema.Gamesbrowser)

	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	_, err = db.Exec(buff.String())

	return err
}
