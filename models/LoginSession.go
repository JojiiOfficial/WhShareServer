package models

import (
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//LoginSession a login session
type LoginSession struct {
	PkID    uint32 `db:"pk_id" orm:"pk,ai"`
	UserID  uint32 `db:"userID"`
	Token   string `db:"sessionToken"`
	Created string `db:"created"`
	IsValid bool   `db:"isValid"`
	User    User
}

//TableLoginSession the table in db for login sessions
const TableLoginSession = "LoginSessions"

//Insert insert a loginSession into the database
func (session *LoginSession) Insert(db *dbhelper.DBhelper) error {
	rs, err := db.Execf("INSERT INTO %s (sessionToken, userID) VALUES(?,?)", []string{TableLoginSession}, session.Token, session.UserID)
	if err != nil {
		return err
	}
	id, err := rs.LastInsertId()
	if err != nil {
		return err
	}
	session.PkID = uint32(id)
	return nil
}

//GetAllSessions returns all valid sessions
func GetAllSessions(db *dbhelper.DBhelper) ([]string, error) {
	var sessions []string
	err := db.QueryRowsf(&sessions, "SELECT sessionToken FROM %s WHERE isValid=1", []string{TableLoginSession})
	return sessions, err
}
