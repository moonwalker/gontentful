package gontentful

import (
	"database/sql"
	"fmt"
)

const (
	createSyncTable = "CREATE TABLE IF NOT EXISTS %s._sync ( id int primary key, token text );"
	insertSyncToken = "INSERT INTO %s._sync (id, token) VALUES (0, '%s') ON CONFLICT (id) DO UPDATE SET token = EXCLUDED.token;"
	selectSyncToken = "SELECT token FROM %s._sync WHERE id = 0;"
)

func GetSyncToken(databaseURL string, schemaName string) (string, error) {
	var syncToken string
	db, _ := sql.Open("postgres", databaseURL)
	row := db.QueryRow(fmt.Sprintf(selectSyncToken, schemaName))
	err := row.Scan(&syncToken)
	if err != nil {
		return "", err
	}
	return syncToken, nil
}

func SaveSyncToken(databaseURL string, schemaName string, token string) error {
	var err error
	db, _ := sql.Open("postgres", databaseURL)
	if schemaName != "" {
		_, err = db.Exec(fmt.Sprintf("SET search_path='%s'", schemaName))
		if err != nil {
			return err
		}
	}
	_, err = db.Exec(fmt.Sprintf(createSyncTable, schemaName))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(insertSyncToken, schemaName, token))
	if err != nil {
		return err
	}
	return nil
}
