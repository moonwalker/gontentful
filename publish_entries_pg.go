package gontentful

import (
	"bytes"
	"text/template"

	"github.com/jmoiron/sqlx"
)

const publishEntriesTemplate = `
	UPDATE {{ .SchemaName }}.{{ .TableName }} 
	SET _status='published',
		_published_at = now(),
		_published_by = {{ .PublishedBy }} 
	WHERE _sys_id IN '{{ .SysIDs }}';`

type PGPublishEntries struct {
	SchemaName  string
	TableName   string
	SysIDs      []string
	PublishedBy string
}

func NewPGPublishEntries(schemaName string, tableName string, sysIDs []string, publishedBy string) *PGPublishEntries {
	return &PGPublishEntries{
		SchemaName:  schemaName,
		TableName:   tableName,
		SysIDs:      sysIDs,
		PublishedBy: publishedBy,
	}
}

func (s *PGPublishEntries) Exec(databaseURL string, txn *sqlx.Tx) error {
	tmpl, err := template.New("").Parse(publishEntriesTemplate)
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

	return err
}
