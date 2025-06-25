package database

import (
	"database/sql"
	"strings"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS data (
        user_address TEXT,
        file_name TEXT,
        cids TEXT,
		UNIQUE(user_address, file_name)
    );`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InsertData(db *sql.DB, userAddress string, fileName string, cids []string) error {
	insertSQL := `INSERT INTO data (user_address, file_name, cids) VALUES (?, ?, ?)`
	_, err := db.Exec(insertSQL, userAddress, fileName, strings.Join(cids, " "))
	return err
}

func QueryData(db *sql.DB, userAddress, filename string) ([]string, error) {
	// Query the database for the CIDs
	querySQL := `SELECT cids FROM data WHERE user_address = ? AND file_name = ?`
	cids := make([]string, 0)
	rows, err := db.Query(querySQL, userAddress, filename)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cidString string
		if err := rows.Scan(&cidString); err != nil {
			return nil, err
		}

		cids = append(cids, strings.Split(cidString, " ")...)
	}

	return cids, nil
}

func ListFiles(db *sql.DB, userAddress string) ([]string, error) {
	// Query the database for file names
	querySQL := `SELECT file_name FROM data WHERE user_address = ?`
	rows, err := db.Query(querySQL, userAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var fileName string
		if err := rows.Scan(&fileName); err != nil {
			return nil, err
		}
		files = append(files, fileName)
	}

	return files, nil
}
