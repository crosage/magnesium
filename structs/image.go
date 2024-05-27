package structs

type Image struct {
	ID       int    `json:"id"`
	PID      int    `json:"pid"`
	Author   Author `json:"author"`
	Tags     []Tag  `json:"tags"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Pages    []Page `json:"pages"`
	FileType string `json:"file_type"`
}
