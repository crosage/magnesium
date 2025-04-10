package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

func GetPixivCookie() (string, error) {
	var cookie sql.NullString
	err := db.QueryRow("SELECT value FROM configuration WHERE key = 'pixiv_cookie'").Scan(&cookie)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return cookie.String, nil
}

func UpdatePixivCookie(newCookie string) error {
	_, err := db.Exec(`
		INSERT INTO configuration (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, "pixiv_cookie", newCookie)
	if err != nil {
		log.Error().Err(err).Msg("更新数据库中的 Pixiv Cookie 失败")
		return err
	}
	log.Info().Msg("数据库中的 Pixiv Cookie 已更新")
	return nil
}
