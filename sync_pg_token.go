package gontentful

import (
	"database/sql"
	"fmt"
)

const (
	createSyncTable = "CREATE TABLE IF NOT EXISTS %s_sync ( id int primary key, token text, created_at timestamp without time zone DEFAULT now() );"
	insertSyncToken = "INSERT INTO %s_sync (id, token) VALUES (0, '%s') ON CONFLICT (id) DO UPDATE SET token = EXCLUDED.token, created_at=now();"
	selectSyncToken = "SELECT token FROM %s_sync WHERE id = 0;"
)

func GetSyncToken(databaseURL string, schemaName string) (string, error) {
	var syncToken string
	db, _ := sql.Open("postgres", databaseURL)
	schemaPrefix := ""
	if schemaName != "" {
		schemaPrefix = fmt.Sprintf("%s.", schemaName)
	}
	row := db.QueryRow(fmt.Sprintf(selectSyncToken, schemaPrefix))
	err := row.Scan(&syncToken)
	if err != nil {
		return "", err
	}
	return syncToken, nil
}

func SaveSyncToken(databaseURL string, schemaName string, token string) error {
	var err error
	db, _ := sql.Open("postgres", databaseURL)

	schemaPrefix := ""
	if len(schemaName) > 0 {
		_, err = db.Exec(fmt.Sprintf("SET search_path='%s'", schemaName))
		if err != nil {
			return err
		}
		schemaPrefix = fmt.Sprintf("%s.", schemaName)
	}

	_, err = db.Exec(fmt.Sprintf(createSyncTable, schemaPrefix))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(insertSyncToken, schemaPrefix, token))
	if err != nil {
		return err
	}

	return nil
}
