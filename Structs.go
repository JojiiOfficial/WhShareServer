package main

import (
	"time"
)

// ------------- Database structs ----------------

//User user in db
type User struct {
	Pkid      int       `db:"pk_id" orm:"pk,ai"`
	Username  string    `db:"username"`
	CreatedAt time.Time `db:"createdAt"`
	IsValid   bool      `db:"isValid"`
}

//Source a webhook source
type Source struct {
	PkID         uint32 `db:"pk_id" orm:"pk,ai"`
	Name         string `db:"name"`
	Secret       string `db:"secret"`
	CreatorID    uint32 `db:"creator"`
	CreationTime string `db:"creationTime"`
	IsPrivate    bool   `db:"private"`
	Creator      User   `db:"-" orm:"-"`
}

//LoginSession a login session
type LoginSession struct {
	PkID    uint32 `db:"pk_id" orm:"pk,ai"`
	UserID  uint32 `db:"userID"`
	Token   string `db:"sessionToken"`
	Created string `db:"created"`
	IsValid bool   `db:"isValid"`
	User    User   `db:"-" orm:"-"`
}

// ------------- REST structs ----------------

//-----> Requests

type sourceAddRequest struct {
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"pass"`
}

//-----> Responses

type loginResponse struct {
	Status string `json:"status"`
	Token  string `json:"token"`
}
