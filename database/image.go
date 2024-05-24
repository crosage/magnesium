package database

import "go_/structs"

func CreateImage(pid int, name string, path string, authorId int) (int, error) {
	result, err := db.Exec("INSERT INTO image(pid,author_id,name,path) VALUES (?,?,?,?)", pid, authorId, name, path)
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
		SELECT id,pid,author_id,name,path
		FROM image
		WHERE pid=?
	`, pid)
	row.Scan(&image.ID, &image.PID, &image.Author.ID, &image.Name, &image.Path)
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
