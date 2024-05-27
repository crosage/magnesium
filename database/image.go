package database

import "go_/structs"

func CreateImage(pid int, name string, path string, authorId int, fileType string) (int, error) {
	result, err := db.Exec("INSERT INTO image(pid,author_id,name,path,file_type) VALUES (?,?,?,?,?)", pid, authorId, name, path, fileType)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}
func GetImageById(pid int) (structs.Image, error) {
	var image structs.Image
	var err error
	row := db.QueryRow(`
		SELECT id,pid,author_id,name,path,file_type
		FROM image
		WHERE pid=?
	`, pid)
	row.Scan(&image.ID, &image.PID, &image.Author.ID, &image.Name, &image.Path, &image.FileType)
	image.Author, err = GetAuthorById(image.Author.ID)
	if err != nil {
		return image, err
	}
	image.Tags, err = GetTagsByPid(pid)
	if err != nil {
		return image, err
	}
	image.Pages, err = GetPageByPid(pid)
	if err != nil {
		return image, err
	}
	return image, nil
}

func GetImagesWithPagination(pageNum int, pageSize int) ([]structs.Image, error) {
	var images []structs.Image
	var err error
	offset := (pageNum - 1) * pageSize
	rows, err := db.Query(`
		SELECT id,pid,author_id,name,path,file_type
		FROM image
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var image structs.Image
		err := rows.Scan(&image.ID, &image.PID, &image.Author.ID, &image.Name, &image.Path, &image.FileType)
		if err != nil {
			return nil, err
		}

		image.Author, err = GetAuthorById(image.Author.ID)
		if err != nil {
			return nil, err
		}

		image.Tags, err = GetTagsByPid(image.PID)
		if err != nil {
			return nil, err
		}

		image.Pages, err = GetPageByPid(image.PID)
		if err != nil {
			return nil, err
		}

		images = append(images, image)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return images, nil
}

func CheckPidExists(pid int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM image WHERE pid=?)"
	err := db.QueryRow(query, pid).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
