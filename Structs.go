package main

// ------------- Database structs ----------------

//User user in db
type User struct {
	Pkid      uint32 `db:"pk_id" orm:"pk,ai"`
	Username  string `db:"username"`
	CreatedAt string `db:"createdAt"`
	IsValid   bool   `db:"isValid"`
}

//Source a webhook source
type Source struct {
	PkID         uint32 `db:"pk_id" orm:"pk,ai"`
	Name         string `db:"name"`
	SourceID     string `db:"sourceID"`
	Description  string `db:"description"`
	Secret       string `db:"secret"`
	CreatorID    uint32 `db:"creator"`
	CreationTime string `db:"creationTime"`
	IsPrivate    bool   `db:"private"`
	Creator      User   `db:"-"`
}

//LoginSession a login session
type LoginSession struct {
	PkID    uint32 `db:"pk_id" orm:"pk,ai"`
	UserID  uint32 `db:"userID"`
	Token   string `db:"sessionToken"`
	Created string `db:"created"`
	IsValid bool   `db:"isValid"`
	User    User
}

//Subscription the subscription a user made
type Subscription struct {
	PkID           uint32 `db:"pk_id" orm:"pk,ai"`
	SubscriptionID string `db:"subscriptionID"`
	UserID         uint32 `db:"subscriber"`
	Source         uint32 `db:"source"`
	CallbackURL    string `db:"callbackURL"`
	Time           string `db:"time"`
	LastTrigger    string `db:"lastTrigger"`
}

// ------------- REST structs ----------------

//-----> Requests

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"pass"`
}

type sourceAddRequest struct {
	Token       string `json:"token"`
	Name        string `json:"name"`
	Description string `json:"descr"`
	Private     bool   `json:"private"`
}

type subscriptionRequest struct {
	Token       string `json:"token"`
	SourceID    string `json:"sid"`
	CallbackURL string `json:"cburl"`
}

type unsubscribeRequest struct {
	SubscriptionID string `json:"sid"`
}

//-----> Responses

type loginResponse struct {
	Status string `json:"status"`
	Token  string `json:"token"`
}

type sourceAddResponse struct {
	Status   string `json:"status"`
	Secret   string `json:"secret"`
	SourceID string `json:"id"`
}

type subscriptionResponse struct {
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	SubscriptionID string `json:"sid"`
	Name           string `json:"name"`
}
