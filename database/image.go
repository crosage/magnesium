package database

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
