package database

func InsertImageTag(imageId int, tagId int) error {
	_, err := db.Exec("INSERT INTO image_tag(image_id,tag_id) VALUES(?,?) ", imageId, tagId)
	if err != nil {
		return err
	}
	return nil
}
