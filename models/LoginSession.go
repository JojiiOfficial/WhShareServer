package models

import (
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//LoginSession a login session
type LoginSession struct {
	PkID         uint32    `db:"pk_id" orm:"pk,ai"`
	UserID       uint32    `db:"userID"`
	Token        string    `db:"sessionToken"`
	Created      time.Time `db:"created"`
	IsValid      bool      `db:"isValid"`
	LastAccessed time.Time `db:"lastAccessed"`
	User         User      `db:"-" orm:"-"`
}

//TableLoginSession the table in db for login sessions
const TableLoginSession = "LoginSessions"

//Insert insert a loginSession into the database
func (session *LoginSession) Insert(db *dbhelper.DBhelper) error {
	_, err := db.Insert(session, &dbhelper.InsertOption{
		TableName:    TableLoginSession,
		SetPK:        true,
		IgnoreFields: []string{"isValid"},
	})

	return err
}

//GetAllSessionTokens returns all valid sessions
func GetAllSessionTokens(db *dbhelper.DBhelper) ([]string, error) {
	var sessions []string
	err := db.QueryRowsf(&sessions, "SELECT sessionToken FROM %s WHERE isValid=1", []string{TableLoginSession})
	return sessions, err
}

//UpdateLastAccessedByToken updates loginsessions last accessed
func (session LoginSession) UpdateLastAccessedByToken(db *dbhelper.DBhelper) {
	db.Execf("UPDATE %s SET lastAccessed=now() WHERE sessionToken=?", []string{TableLoginSession}, session.Token)
}
