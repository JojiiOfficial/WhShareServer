package models

//CredentialRequest request containing credentials
type CredentialRequest struct {
	Username string `json:"username"`
	Password string `json:"pass"`
}

//SourceAddRequest request to create a source
type SourceAddRequest struct {
	Token       string `json:"token"`
	Name        string `json:"name"`
	Description string `json:"descr"`
	Private     bool   `json:"private"`
	Mode        uint8  `json:"mode"`
}

//SubscriptionRequest request to subscribe
type SubscriptionRequest struct {
	Token       string `json:"token"`
	SourceID    string `json:"sid"`
	CallbackURL string `json:"cbUrl"`
}

//UnsubscribeRequest request for unsubscribing a source
type UnsubscribeRequest struct {
	SubscriptionID string `json:"sid"`
}

//SubscriptionUpdateCallbackRequest request for updating callback
type SubscriptionUpdateCallbackRequest struct {
	Token          string `json:"token"`
	SubscriptionID string `json:"subID"`
	CallbackURL    string `json:"cbUrl"`
}

//SourceRequest request containing sourceData
type SourceRequest struct {
	Token    string `json:"token"`
	SourceID string `json:"sid,omitempty"`
	Content  string `json:"content,omitempty"`
}

//TokenOnlyRequest a request containing only a token
type TokenOnlyRequest struct {
	Token string `json:"token"`
}
