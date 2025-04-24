package structs

type Image struct {
	ID            int       `json:"id"`
	PID           int       `json:"pid"`
	Author        Author    `json:"author"`
	Name          string    `json:"name"`
	BookmarkCount int       `json:"bookmark_count"`
	IsBookmarked  bool      `json:"is_bookmarked"`
	Local         bool      `json:"local"`
	URLs          ImageURLs `json:"urls"`
	Tags          []Tag     `json:"tags"`
	Pages         []Page    `json:"pages"`
}
