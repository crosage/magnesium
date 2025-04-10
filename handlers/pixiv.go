package handlers

type BookmarkPayload struct {
	IllustID string   `json:"illust_id"`
	Restrict int      `json:"restrict"`
	Comment  string   `json:"comment"`
	Tags     []string `json:"tags"`
}
