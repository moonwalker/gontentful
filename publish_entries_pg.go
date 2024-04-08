package gontentful

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

const publishEntriesTemplate = `
	UPDATE {{ .SchemaName }}.{{ .TableName }} 
	SET _status='published',
		_version = _version + 1,
		_published_at = to_timestamp('{{ .PublishedAt }}','YYYY-MM-DDThh24:mi:ssZ'),
		_published_by = '{{ .PublishedBy }}'
	WHERE _sys_id IN('{{ .SysIDs }}');`

type PGPublishEntries struct {
	SchemaName  string
	TableName   string
	SysIDs      []string
	PublishedAt string
	PublishedBy string
}

func NewPGPublishEntries(schemaName string, tableName string, sysIDs []string, publishedBy string, publishedAt string) *PGPublishEntries {
	return &PGPublishEntries{
		SchemaName:  schemaName,
		TableName:   tableName,
		SysIDs:      sysIDs,
		PublishedAt: publishedAt,
		PublishedBy: publishedBy,
	}
}

func (s *PGPublishEntries) Exec(databaseURL string) error {
	str, err := s.Render()
	if err != nil {
		return err
	}

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

	if s.SchemaName != "" {
		// set schema name
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.SchemaName))
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

func (s *PGPublishEntries) Render() (string, error) {
	tmpl, err := template.New("").Parse(publishEntriesTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return "", err
	}

	// fmt.Println(buff.String())

	return buff.String(), nil
}
