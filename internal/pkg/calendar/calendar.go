package calendar

// Response represents a Calendar response.
type Response struct {
	Days []Day `json:"days"`
}

// Day represents a calendar day containing items.
type Day struct {
	TodayID string `json:"today_id"`
	Items   []Item `json:"items"`
}

// Item represents the information of a calendar item.
type Item struct {
	Details string `json:"details"`
	Status  string `json:"status"`
	Type    string `json:"type"`
}
