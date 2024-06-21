package database

import (
	"database/sql"
	"go_/structs"
)

func GetOrCreateTagIdByName(name string) (int, error) {
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
func GetTags(page int, size int) ([]structs.Tag, error) {
	offset := (page - 1) * size
	rows, err := db.Query(`
		SELECT id, name, translate_name 
		FROM tag 
		ORDER BY id 
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []structs.Tag
	for rows.Next() {
		var tag structs.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.TranslateName); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, nil
}
