package link

type Link struct {
	Code        string `json:"code"`
	OriginalURL string `json:"originalurl"`
	Clicks      int    `json:"clicks"`
}
