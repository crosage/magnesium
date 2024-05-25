package database

import (
	"github.com/rs/zerolog/log"
	"go_/structs"
)

func GetAllGalleries() ([]structs.LocalGallery, error) {
	var galleries []structs.LocalGallery
	rows, err := db.Query("SELECT id,path FROM local_gallery")
	if err != nil {
		log.Error().Err(err).Msg("查询所有gallery时出现错误")
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var gallery structs.LocalGallery
		if err := rows.Scan(&gallery.ID, &gallery.Path); err != nil {
			log.Error().Err(err).Msg("为gallery赋值出现错误")
			return nil, err
		}
		galleries = append(galleries, gallery)
	}
	return galleries, nil
}

func GetGalleryById(id int) (structs.LocalGallery, error) {
	var gallery structs.LocalGallery
	row := db.QueryRow("SELECT id,path FROM local_gallery WHERE id=?", id)
	err := row.Scan(&gallery.ID, &gallery.Path)
	return gallery, err
}
func CreateLocalGalleryPath(path string) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM local_gallery WHERE path = ?", path).Scan(&count)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("Error checking for existing gallery path")
		return err
	}

	if count > 0 {
		log.Info().Str("path", path).Msg("Gallery path already exists, not inserting")
		return nil
	}

	_, err = db.Exec("INSERT INTO local_gallery (path) VALUES(?)", path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("Error inserting gallery path")
		return err
	}

	return nil
}
func DeleteLocalGalleryByID(id int) error {
	result, err := db.Exec("DELETE FROM local_gallery WHERE id = ?", id)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("删除 gallery 时出现错误")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("获取影响行数时出现错误")
		return err
	}

	if rowsAffected == 0 {
		log.Warn().Int("id", id).Msg("没有找到要删除的 gallery")
		return nil
	}

	log.Info().Int("id", id).Msg("成功删除 gallery")
	return nil
}
