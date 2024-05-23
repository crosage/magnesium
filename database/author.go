package database

import (
	"database/sql"
	"go_/structs"
)

func CreateAuthor(author structs.Author) error {
	stmt, err := db.Prepare("INSERT INTO author (name,uid)  VALUES (?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(author.Name, author.UID)
	if err != nil {
		return err
	}
	return nil
}
func GetAuthorByName(name string) (structs.Author, error) {
	var author structs.Author
	row := db.QueryRow("SELECT name,uid from author where name=?", name)
	err := row.Scan(&author.Name, &author.UID)
	if err != nil {
		if err == sql.ErrNoRows {
			return author, sql.ErrNoRows
		}
		return author, err
	}
	return author, nil
}
