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

func SearchImages(tags []string, pageNum int, pageSize int, authorName string, sortBy string, sortOrder string) ([]structs.Image, int, error) {
	var images []structs.Image
	var count int

	query, args := buildQuery(tags, pageNum, pageSize, authorName, sortBy, sortOrder)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	query, args = buildCountQuery(tags, authorName)
	err = db.QueryRow(query, args...).Scan(&count)
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

func GetImagesWithPagination(pageNum int, pageSize int, authorName string, sortBy string, sortOrder string) ([]structs.Image, int, error) {
	var images []structs.Image
	var err error
	var count int

	offset := (pageNum - 1) * pageSize

	var sb strings.Builder
	sb.WriteString(`
		SELECT i.id, i.pid, i.author_id, i.name, i.path, i.file_type
		FROM image i
	`)

	if authorName != "" {
		sb.WriteString("JOIN author a ON i.author_id = a.id ")
	}

	if authorName != "" {
		sb.WriteString(fmt.Sprintf("WHERE a.name = '%s' ", authorName))
	}

	if sortBy != "" {
		sb.WriteString(fmt.Sprintf("ORDER BY %s %s ", sortBy, sortOrder))
	} else {
		sb.WriteString(fmt.Sprintf("ORDER BY i.pid %s", sortOrder))
	}

	sb.WriteString(fmt.Sprintf("LIMIT %d OFFSET %d", pageSize, offset))

	rows, err := db.Query(sb.String())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	countQuery := `
		SELECT COUNT(*)
		FROM image i
	`
	if authorName != "" {
		countQuery += "JOIN author a ON i.author_id = a.id WHERE a.name = '" + authorName + "'"
	}
	err = db.QueryRow(countQuery).Scan(&count)
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

func buildQuery(tags []string, page int, pageSize int, authorName string, sortBy string, sortOrder string) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}

	sb.WriteString("SELECT i.id, i.pid, i.author_id, i.name, i.path, i.file_type ")
	sb.WriteString("FROM image i ")
	sb.WriteString("JOIN image_tag it ON i.pid = it.image_id ")
	sb.WriteString("JOIN tag t ON it.tag_id = t.id ")
	if authorName != "" {
		sb.WriteString("JOIN author a ON i.author_id = a.id ")
	}
	sb.WriteString("WHERE t.name IN (")
	for i, tag := range tags {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("?")
		args = append(args, tag)
	}
	sb.WriteString(") ")
	if authorName != "" {
		sb.WriteString("AND a.name = ? ")
		args = append(args, authorName)
	}
	sb.WriteString("GROUP BY i.id ")
	sb.WriteString("HAVING COUNT(DISTINCT t.id) = ? ")
	args = append(args, len(tags))

	if sortBy == "" {
		sortBy = "i.pid"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	sb.WriteString("ORDER BY ")
	sb.WriteString(sortBy)
	sb.WriteString(" ")
	sb.WriteString(sortOrder)
	sb.WriteString(" ")
	offset := (page - 1) * pageSize
	limit := pageSize
	sb.WriteString("LIMIT ? OFFSET ?")
	args = append(args, limit, offset)

	fmt.Println("Query:", sb.String())
	fmt.Println("Args:", args)
	return sb.String(), args
}

func buildCountQuery(tags []string, authorName string) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	sb.WriteString("SELECT COUNT(*) FROM (")
	sb.WriteString("SELECT i.id ")
	sb.WriteString("FROM image i ")
	sb.WriteString("JOIN image_tag it ON i.pid = it.image_id ")
	sb.WriteString("JOIN tag t ON it.tag_id = t.id ")

	if authorName != "" {
		sb.WriteString("JOIN author a ON i.author_id = a.id ")
	}

	sb.WriteString("WHERE t.name IN (")
	for i, tag := range tags {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("?")
		args = append(args, tag)
	}
	sb.WriteString(") ")
	if authorName != "" {
		sb.WriteString("AND a.name = ? ")
		args = append(args, authorName)
	}
	sb.WriteString("GROUP BY i.id ")
	sb.WriteString("HAVING COUNT(DISTINCT t.id) = ? ")
	sb.WriteString(") AS subquery")
	args = append(args, len(tags))

	fmt.Println("Count Query:", sb.String())
	fmt.Println("Args:", args)
	return sb.String(), args
}
