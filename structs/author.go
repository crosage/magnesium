package structs

type Author struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}

type AuthorCount struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	UID   string `json:"uid"`
	Count int    `json:"count"`
}
