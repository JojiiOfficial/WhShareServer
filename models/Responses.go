package models

//LoginResponse response for login
type LoginResponse struct {
	Token string `json:"token"`
}

//SourceAddResponse response for adding sources
type SourceAddResponse struct {
	Secret   string `json:"secret"`
	SourceID string `json:"id"`
}

//SubscriptionResponse response for subscription
type SubscriptionResponse struct {
	Message        string `json:"message,omitempty"`
	SubscriptionID string `json:"sid"`
	Name           string `json:"name"`
	Mode           uint8  `json:"mode"`
}

//ListSourcesResponse response containing a list of sources
type ListSourcesResponse struct {
	Sources []Source `json:"sources,omitempty"`
}
