package database

import (
	"go_/structs"
)

func InsertPageByPid(imageID, pageNum int) (int, error) {
	result, err := db.Exec("INSERT INTO page (image_id, page_num) VALUES (?, ?)", imageID, pageNum)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func GetPageByPid(pid int) ([]structs.Page, error) {
	var pages []structs.Page
	rows, err := db.Query(`
		SELECT p.id,p.image_id,p.page_id
		FROM page p 
		INNER JOIN image i ON p.image_id=i.pid
		WHERE i.pid=?
	`, pid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var page structs.Page
		if err := rows.Scan(&page.ID, &page.ImageID, &page.PageID); err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func UpdatePage(id, newPageNum int) error {
	_, err := db.Exec("UPDATE page SET page_num = ? WHERE id = ?", newPageNum, id)
	return err
}

func DeletePage(id int) error {
	_, err := db.Exec("DELETE FROM page WHERE id = ?", id)
	return err
}
