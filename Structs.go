package main

import "time"

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
	CreatorID    uint32 `db:"creator"`
	Creator      User   `db:"-" orm:"-"`
	Name         string `db:"name"`
	Secret       string `db:"secret"`
	CreationTime string `db:"creationTime"`
	IsPrivate    bool   `db:"private"`
}
