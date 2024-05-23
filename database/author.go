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
func GetOrCreateAuthor(author structs.Author) (structs.Author, error) {
	author, err := GetAuthorByName(author.Name)
	if err == nil {
		return author, nil
	} else if err == sql.ErrNoRows {
		newAuthor := structs.Author{Name: author.Name, UID: author.UID}
		err = CreateAuthor(newAuthor)
		if err != nil {
			return structs.Author{}, err
		}
		return newAuthor, nil
	} else {
		return structs.Author{}, err
	}
}
