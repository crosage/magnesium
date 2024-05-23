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
func AddLocalGalleryPath(path string) error {
	_, err := db.Exec("INSERT INTO local_gallery (path ) VALUES(?) ", path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("Error inserting gallery path")
		return err
	}
	return nil
}
