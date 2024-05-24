package database

import (
	"database/sql"
	"go_/structs"
)

func CreateAuthor(author structs.Author) (int, error) {
	result, err := db.Exec("INSERT INTO author (name, uid) VALUES (?, ?)", author.Name, author.UID)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
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
func GetAuthorById(id int) (structs.Author, error) {
	var author structs.Author
	row := db.QueryRow("SELECT name,uid from author where id=?", id)
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
	existingAuthor, err := GetAuthorByName(author.Name)
	if err == nil {
		return existingAuthor, nil
	} else if err == sql.ErrNoRows {
		newAuthor := structs.Author{Name: author.Name, UID: author.UID}
		id, err := CreateAuthor(newAuthor)
		newAuthor.ID = id
		if err != nil {
			return structs.Author{}, err
		}
		return newAuthor, nil
	} else {
		return structs.Author{}, err
	}
}
