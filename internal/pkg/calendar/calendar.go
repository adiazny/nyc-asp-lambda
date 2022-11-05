package calendar

type Response struct {
	Days []Day `json:"days"`
}

type Day struct {
	TodayID string `json:"today_id"`
	Items   []Item `json:"items"`
}

type Item struct {
	Details string `json:"details"`
	Status  string `json:"status"`
	Type    string `json:"type"`
}
