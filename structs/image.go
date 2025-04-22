package structs

type Image struct {
	ID            int
	PID           int
	Author        Author
	Name          string
	BookmarkCount int
	IsBookmarked  bool
	Local         bool
	URLs          ImageURLs
	Tags          []Tag
	Pages         []Page
}
