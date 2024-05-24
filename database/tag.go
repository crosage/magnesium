package database

import "database/sql"

func GetorCreateTagIdByName(name string) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM tag WHERE name = ?", name).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			result, err := db.Exec("INSERT INTO tag (name) VALUES (?)", name)
			if err != nil {
				return 0, err
			}
			insertedID, err := result.LastInsertId()
			if err != nil {
				return 0, err
			}
			id = int(insertedID)
		} else {
			return 0, err
		}
	}

	return id, nil
}
