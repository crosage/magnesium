package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

var db *sql.DB

func InitDatabase() {
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal().Err(err).Msg("Fail to open database")
	}
	createTables()
}
func createTables() {
	// 创建Lib表
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS local_gallery(
	    id INTEGER PRIMARY KEY,
		path Text
	)
	`)
	if err != nil {
		log.Fatal().Err(err)
	}
	// 创建Author表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS author (
		id INTEGER PRIMARY KEY,
		name TEXT,
		uid TEXT
	);`)
	if err != nil {
		log.Fatal().Err(err)
	}
	// 创建Tag表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS tag (
		id INTEGER PRIMARY KEY,
		name TEXT,
		translate_name TEXT DEFAULT ""
	);`)
	if err != nil {
		log.Fatal().Err(err)
	}

	// 创建Image表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS image (
		id INTEGER PRIMARY KEY,
		pid INTEGER,
		author_id INTEGER,
		name TEXT,
		path TEXT,
		file_type TEXT,
		FOREIGN KEY (author_id) REFERENCES author(id)  ON DELETE CASCADE
	);`)
	if err != nil {
		log.Fatal().Err(err)
	}

	// 创建Page表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS page (
		id INTEGER PRIMARY KEY,
		image_id INTEGER,
		page_id INTEGER,
		FOREIGN KEY (image_id) REFERENCES image(id) ON DELETE CASCADE 
	);`)
	if err != nil {
		log.Fatal().Err(err)
	}

	// 创建ImageTag中间表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS image_tag (
		id INTEGER PRIMARY KEY,
		image_id INTEGER,
		tag_id INTEGER,
		FOREIGN KEY (image_id) REFERENCES image(pid) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tag(id) ON DELETE CASCADE
	);`)
	if err != nil {
		log.Fatal().Err(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS configuration (
		id INTEGER PRIMARY KEY, -- Or SERIAL PRIMARY KEY for PostgreSQL, INT AUTO_INCREMENT PRIMARY KEY for MySQL
		key TEXT UNIQUE NOT NULL,
		value TEXT
	);`)
	if err != nil {
		log.Fatal().Err(err)
	}
}
