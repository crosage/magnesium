package database

import "go_/structs"

func InsertImageTag(pid int, tagId int) error {
	_, err := db.Exec("INSERT INTO image_tag(image_id,tag_id) VALUES(?,?) ", pid, tagId)
	if err != nil {
		return err
	}
	return nil
}

func GetTagsByPid(pid int) ([]structs.Tag, error) {
	var tags []structs.Tag
	rows, err := db.Query(`
		SELECT t.id,t.name,t.translate_name
		FROM tag t
		INNER JOIN image_tag it ON t.id=it.tag_id
		WHERE it.image_id=?
	`, pid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag structs.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.TranslateName); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}
