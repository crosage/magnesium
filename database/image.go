package database

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"go_/structs"
	"strings"
	"time"
)

func CreateImage(pid int, name string, authorId int, bookmarkCount int, isBookmarked bool, urls structs.ImageURLs) (int, error) {
	nowUnix := time.Now().Unix()
	result, err := db.Exec(`
        INSERT INTO image(pid, author_id, name, url_original,url_mini, url_thumb, url_small, url_regular,updated_at,bookmark_count,is_bookmarked)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?,?,?,?)`,
		pid, authorId, name, urls.Original, urls.Mini, urls.Thumb, urls.Small, urls.Regular, nowUnix, bookmarkCount, isBookmarked)
	if err != nil {
		return 0, fmt.Errorf("failed to execute insert for pid %d: %w", pid, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id for pid %d: %w", pid, err)
	}
	return int(id), nil
}

func UpdateImage(pid int, name string, authorId int, bookmarkCount int, isBookmarked bool, urls structs.ImageURLs) error {
	nowUnix := time.Now().Unix()
	result, err := db.Exec(`
        UPDATE image
        SET author_id = ?, name = ?, url_original = ?, url_mini = ?,
            url_thumb = ?, url_small = ?, url_regular = ?, updated_at = ?, bookmark_count = ?,is_bookmarked=?
        WHERE pid = ?`,
		authorId, name, urls.Original, urls.Mini, urls.Thumb, urls.Small, urls.Regular, nowUnix, bookmarkCount, isBookmarked, pid)
	if err != nil {
		return fmt.Errorf("failed to execute update for pid %d: %w", pid, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Warning: could not get rows affected for update pid %d: %v\n", pid, err)
		return nil
	}
	if rowsAffected == 0 {
		return fmt.Errorf("update failed for pid %d: record not found (or data was identical)", pid)
	}

	return nil
}

func GetImageById(pid int) (structs.Image, error) {
	var image structs.Image
	var err error

	row := db.QueryRow(`
       SELECT id, pid, author_id, name, bookmark_count, is_bookmarked, local,
              url_original, url_mini, url_thumb, url_small, url_regular
       FROM image
       WHERE pid = ?
    `, pid)

	err = row.Scan(
		&image.ID, &image.PID, &image.Author.ID, &image.Name,
		&image.BookmarkCount, &image.IsBookmarked, &image.Local,
		&image.URLs.Original, &image.URLs.Mini, &image.URLs.Thumb, &image.URLs.Small, &image.URLs.Regular,
	)
	if err != nil {
		return image, err
	}

	image.Author, err = GetAuthorById(image.Author.ID)
	if err != nil {
		return image, fmt.Errorf("failed to get author %d for image %d: %w", image.Author.ID, pid, err)
	}
	image.Tags, err = GetTagsByPid(pid)
	if err != nil {
		return image, fmt.Errorf("failed to get tags for image %d: %w", pid, err)
	}
	image.Pages, err = GetPageByPid(pid)
	if err != nil {
		return image, fmt.Errorf("failed to get pages for image %d: %w", pid, err)
	}
	return image, nil
}

func GetAuthorImageCounts() ([]structs.AuthorCount, error) {
	var authorImageCounts []structs.AuthorCount

	rows, err := db.Query(`
        SELECT a.id, a.name, COUNT(i.id) AS image_count
        FROM author a
        INNER JOIN image i ON a.id = i.author_id
        GROUP BY a.id, a.name
        ORDER BY image_count DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var authorImageCount structs.AuthorCount
		err := rows.Scan(&authorImageCount.ID, &authorImageCount.Name, &authorImageCount.Count)
		if err != nil {
			return nil, err
		}
		authorImageCounts = append(authorImageCounts, authorImageCount)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return authorImageCounts, nil
}

var allowedSortColumns = map[string]bool{
	"id":             true,
	"pid":            true,
	"name":           true,
	"bookmark_count": true,
}

var allowedSortOrders = map[string]bool{
	"ASC":  true,
	"DESC": true,
}

func SearchImages(tags []string, pageNum int, pageSize int, authorName string, sortBy string, sortOrder string, minBookmarkCount *int, maxBookmarkCount *int, isBookmarked *bool) ([]structs.Image, int, error) {
	var images []structs.Image
	var count int

	query, args := buildQuery(tags, pageNum, pageSize, authorName, sortBy, sortOrder, minBookmarkCount, maxBookmarkCount, isBookmarked)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	countQuery, countArgs := buildCountQuery(tags, authorName, minBookmarkCount, maxBookmarkCount, isBookmarked)
	log.Debug().Str("query", countQuery).Interface("args", countArgs).Msg("Executing SearchImages count query") // Debug 日志

	err = db.QueryRow(countQuery, countArgs...).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []structs.Image{}, 0, nil
		}
		return nil, 0, err
	}

	for rows.Next() {
		var image structs.Image
		err = rows.Scan(
			&image.ID, &image.PID, &image.Author.ID, &image.Name,
			&image.BookmarkCount, &image.IsBookmarked, &image.Local,
			&image.URLs.Original, &image.URLs.Mini, &image.URLs.Thumb, &image.URLs.Small, &image.URLs.Regular,
		)
		if err != nil {
			return nil, 0, err
		}

		image.Author, err = GetAuthorById(image.Author.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get author %d during search: %w", image.Author.ID, err)
		}
		image.Tags, err = GetTagsByPid(image.PID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get tags for pid %d during search: %w", image.PID, err)
		}
		image.Pages, err = GetPageByPid(image.PID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get pages for pid %d during search: %w", image.PID, err)
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
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}
func buildQuery(tags []string, page int, pageSize int, authorName string, sortBy string, sortOrder string, minBookmarkCount *int, maxBookmarkCount *int, isBookmarked *bool) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	var whereConditions []string
	var joinClauses []string

	sb.WriteString(`SELECT DISTINCT i.id, i.pid, i.author_id, i.name, i.bookmark_count, i.is_bookmarked, i.local,
                       i.url_original, i.url_mini, i.url_thumb, i.url_small, i.url_regular `)

	sb.WriteString(" FROM image i ")

	hasTags := len(tags) > 0

	if authorName != "" {
		joinClauses = append(joinClauses, " JOIN author a ON i.author_id = a.id ")
	}

	if hasTags {
		joinClauses = append(joinClauses, " JOIN image_tag it ON i.pid = it.image_id ")
		joinClauses = append(joinClauses, " JOIN tag t ON it.tag_id = t.id ")
	}

	whereConditions = append(whereConditions, "i.url_regular IS NOT NULL")

	if authorName != "" {
		whereConditions = append(whereConditions, "a.name = ?")
		args = append(args, authorName)
	}

	if minBookmarkCount != nil {
		whereConditions = append(whereConditions, "i.bookmark_count >= ?")
		args = append(args, *minBookmarkCount)
	}
	if maxBookmarkCount != nil {
		whereConditions = append(whereConditions, "i.bookmark_count <= ?")
		args = append(args, *maxBookmarkCount)
	}

	if isBookmarked != nil {
		whereConditions = append(whereConditions, "i.is_bookmarked = ?")
		args = append(args, *isBookmarked)
	}

	if hasTags {
		whereConditions = append(whereConditions, "t.name IN ("+strings.Repeat("?,", len(tags)-1)+"?)")
		for _, tag := range tags {
			args = append(args, tag)
		}
	}

	if len(joinClauses) > 0 {
		sb.WriteString(strings.Join(joinClauses, ""))
	}

	if len(whereConditions) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(whereConditions, " AND "))
	}

	if hasTags {
		sb.WriteString(` GROUP BY i.id, i.pid, i.author_id, i.name, i.bookmark_count, i.is_bookmarked, i.local,
                               i.url_original, i.url_mini, i.url_thumb, i.url_small, i.url_regular `)
		sb.WriteString(" HAVING COUNT(DISTINCT t.id) = ? ")
		args = append(args, len(tags))
	}

	dbSortColumn := "i.pid"
	safeSortBy, sortByOK := allowedSortColumns[sortBy]
	if sortByOK && safeSortBy {
		if sortBy == "name" || sortBy == "id" || sortBy == "pid" || sortBy == "bookmark_count" {
			dbSortColumn = "i." + sortBy
		}
	}

	dbSortOrder := "DESC"
	safeSortOrder, orderOK := allowedSortOrders[strings.ToUpper(sortOrder)]
	if orderOK && safeSortOrder {
		dbSortOrder = strings.ToUpper(sortOrder)
	}
	sb.WriteString(fmt.Sprintf(" ORDER BY %s %s ", dbSortColumn, dbSortOrder))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	sb.WriteString(" LIMIT ? OFFSET ? ")
	args = append(args, pageSize, offset)

	return sb.String(), args
}

func buildCountQuery(tags []string, authorName string, minBookmarkCount *int, maxBookmarkCount *int, isBookmarked *bool) (string, []interface{}) {
	var countSb strings.Builder
	var args []interface{}
	var whereConditions []string
	var joinClauses []string

	hasTags := len(tags) > 0

	if hasTags {
		countSb.WriteString("SELECT COUNT(*) FROM (SELECT 1 FROM image i ")
	} else {
		countSb.WriteString("SELECT COUNT(i.id) FROM image i ")
	}

	if authorName != "" {
		joinClauses = append(joinClauses, " JOIN author a ON i.author_id = a.id ")
	}
	if hasTags {
		joinClauses = append(joinClauses, " JOIN image_tag it ON i.pid = it.image_id ")
		joinClauses = append(joinClauses, " JOIN tag t ON it.tag_id = t.id ")
	}

	if len(joinClauses) > 0 {
		if hasTags {
			countSb.WriteString(strings.Join(joinClauses, ""))
		} else {
			countSb.WriteString(strings.Join(joinClauses, ""))
		}
	}

	whereConditions = append(whereConditions, "i.url_regular IS NOT NULL")

	if authorName != "" {
		whereConditions = append(whereConditions, "a.name = ?")
		args = append(args, authorName)
	}
	if minBookmarkCount != nil {
		whereConditions = append(whereConditions, "i.bookmark_count >= ?")
		args = append(args, *minBookmarkCount)
	}
	if maxBookmarkCount != nil {
		whereConditions = append(whereConditions, "i.bookmark_count <= ?")
		args = append(args, *maxBookmarkCount)
	}
	if isBookmarked != nil {
		whereConditions = append(whereConditions, "i.is_bookmarked = ?")
		args = append(args, *isBookmarked)
	}
	if hasTags {
		whereConditions = append(whereConditions, "t.name IN ("+strings.Repeat("?,", len(tags)-1)+"?)")
		for _, tag := range tags {
			args = append(args, tag)
		}
	}

	if len(whereConditions) > 0 {
		countSb.WriteString(" WHERE ")
		countSb.WriteString(strings.Join(whereConditions, " AND "))
	}

	if hasTags {
		countSb.WriteString(" GROUP BY i.id ")
		countSb.WriteString(" HAVING COUNT(DISTINCT t.id) = ? ")
		args = append(args, len(tags))
		countSb.WriteString(") AS matching_images")
	}

	return countSb.String(), args
}

func GetAllPids() ([]int, error) {
	rows, err := db.Query(`SELECT pid FROM image ORDER BY pid`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pids from image table: %w", err)
	}
	defer rows.Close()

	var pids []int
	for rows.Next() {
		var pid int
		if err := rows.Scan(&pid); err != nil {
			fmt.Printf("Warning: failed to scan pid: %v\n", err)
			continue
		}
		pids = append(pids, pid)
	}

	if err = rows.Err(); err != nil {
		return pids, fmt.Errorf("error encountered during row iteration for pids: %w", err)
	}

	return pids, nil
}

func GetPidsByBookmarkRange(minBookmarks int, maxBookmarks int) ([]int, error) {
	query := `
        SELECT pid 
        FROM image 
        WHERE bookmark_count >= ? AND bookmark_count <= ? 
        ORDER BY pid
    `

	rows, err := db.Query(query, minBookmarks, maxBookmarks)
	if err != nil {
		return nil, fmt.Errorf("failed to query pids by bookmark range [%d, %d]: %w", minBookmarks, maxBookmarks, err)
	}
	defer rows.Close()
	var pids []int
	for rows.Next() {
		var pid int
		if err := rows.Scan(&pid); err != nil {
			fmt.Printf("Warning: failed to scan pid during bookmark range query: %v\n", err)
			continue
		}
		pids = append(pids, pid)
	}
	if err = rows.Err(); err != nil {
		return pids, fmt.Errorf("error encountered during row iteration for bookmark range query [%d, %d]: %w", minBookmarks, maxBookmarks, err)
	}
	return pids, nil
}
