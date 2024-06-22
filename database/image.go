package database

import (
	"fmt"
	"go_/structs"
	"strings"
)

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

func SearchImages(tags []string, pageNum int, pageSize int) ([]structs.Image, int, error) {
	var images []structs.Image
	var count int

	query, args := buildQuery(tags, pageNum, pageSize)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	query, args = buildCountQuery(tags)
	err = db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return nil, 0, err
	}
	fmt.Println("#########")
	fmt.Println(count)
	fmt.Println("#########")
	for rows.Next() {
		var image structs.Image
		err := rows.Scan(&image.ID, &image.PID, &image.Author.ID, &image.Name, &image.Path, &image.FileType)
		if err != nil {
			return nil, 0, err
		}

		image.Author, err = GetAuthorById(image.Author.ID)
		if err != nil {
			return nil, 0, err
		}

		image.Tags, err = GetTagsByPid(image.PID)
		if err != nil {
			return nil, 0, err
		}

		image.Pages, err = GetPageByPid(image.PID)
		if err != nil {
			return nil, 0, err
		}

		images = append(images, image)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return images, count, nil
}

func GetImagesWithPagination(pageNum int, pageSize int) ([]structs.Image, int, error) {
	var images []structs.Image
	var err error
	var count int

	offset := (pageNum - 1) * pageSize
	rows, err := db.Query(`
		SELECT id,pid,author_id,name,path,file_type
		FROM image
		ORDER BY pid
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM image
	`).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	for rows.Next() {
		var image structs.Image
		err := rows.Scan(&image.ID, &image.PID, &image.Author.ID, &image.Name, &image.Path, &image.FileType)
		if err != nil {
			return nil, 0, err
		}

		image.Author, err = GetAuthorById(image.Author.ID)
		if err != nil {
			return nil, 0, err
		}

		image.Tags, err = GetTagsByPid(image.PID)
		if err != nil {
			return nil, 0, err
		}

		image.Pages, err = GetPageByPid(image.PID)
		if err != nil {
			return nil, 0, err
		}

		images = append(images, image)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return images, count, nil
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

func buildQuery(tags []string, page int, pageSize int) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}

	sb.WriteString("SELECT i.id, i.pid, i.author_id, i.name, i.path, i.file_type ")
	sb.WriteString("FROM image i ")
	sb.WriteString("JOIN image_tag it ON i.pid = it.image_id ")
	sb.WriteString("JOIN tag t ON it.tag_id = t.id ")
	sb.WriteString("WHERE t.name IN (")
	for i, tag := range tags {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("?")
		args = append(args, tag)
	}
	sb.WriteString(") ")
	sb.WriteString("GROUP BY i.id ")
	sb.WriteString("HAVING COUNT(DISTINCT t.id) = ? ")
	args = append(args, len(tags))
	sb.WriteString("ORDER BY i.pid ")
	offset := (page - 1) * pageSize
	limit := pageSize
	sb.WriteString("LIMIT ? OFFSET ?")
	args = append(args, limit, offset)
	fmt.Println("Query:", sb.String())
	fmt.Println("Args:", args)
	return sb.String(), args
}

func buildCountQuery(tags []string) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	sb.WriteString("SELECT COUNT(*) FROM (")
	sb.WriteString("SELECT i.id ")
	sb.WriteString("FROM image i ")
	sb.WriteString("JOIN image_tag it ON i.pid = it.image_id ")
	sb.WriteString("JOIN tag t ON it.tag_id = t.id ")
	sb.WriteString("WHERE t.name IN (")
	for i, tag := range tags {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("?")
		args = append(args, tag)
	}
	sb.WriteString(") ")
	sb.WriteString("GROUP BY i.id ")
	sb.WriteString("HAVING COUNT(DISTINCT t.id) = ? ")
	sb.WriteString(") AS subquery")
	args = append(args, len(tags))

	fmt.Println("Count Query:", sb.String())
	fmt.Println("Args:", args)
	return sb.String(), args
}
