package database

import (
	"go_/structs"
)

func InsertPageByPid(pid int, pageId int) (int, error) {
	// 检查记录是否已经存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM page WHERE image_id = ? AND page_id = ?)", pid, pageId).Scan(&exists)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, nil
	}

	// 如果记录不存在，则插入新记录
	result, err := db.Exec("INSERT INTO page (image_id, page_id) VALUES (?, ?)", pid, pageId)
	if err != nil {
		return 0, err
	}

	// 获取最后插入的ID
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
