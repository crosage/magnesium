package structs

type Image struct {
	ID     int
	PID    int
	Page   int
	Author Author
	Tags   []Tag
	Name   string
	Path   string
	Pages  []Page
}
