package database

import (
	"database/sql"
	"fmt"
	"go_/structs"
	"sort"
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

func GetTagCounts() ([]structs.TagCount, error) {

	query := `
        SELECT tag.id,tag.name, COUNT(image_tag.tag_id) as count
        FROM tag
        INNER JOIN image_tag ON tag.id = image_tag.tag_id
        GROUP BY tag.name;
    `
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tagCounts []structs.TagCount
	for rows.Next() {
		var tagCount structs.TagCount
		if err := rows.Scan(&tagCount.ID, &tagCount.Name, &tagCount.Count); err != nil {
			return nil, err
		}
		tagCounts = append(tagCounts, tagCount)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(tagCounts, func(i, j int) bool {
		return tagCounts[i].Count > tagCounts[j].Count
	})

	return tagCounts, nil
}

func DeleteImageTags(pid int) error {
	_, err := db.Exec(`DELETE FROM image_tag WHERE image_id = ?`, pid)
	if err != nil {
		return fmt.Errorf("failed to delete tags for pid %d: %w", pid, err)
	}
	return nil
}
