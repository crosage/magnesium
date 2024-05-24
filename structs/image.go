package structs

type Image struct {
	ID     int
	PID    int
	Author Author
	Tags   []Tag
	Name   string
	Path   string
	Pages  []Page
}
