package main

import dbhelper "github.com/JojiiOfficial/GoDBHelper"

//User user in db
type User struct {
	Pkid       uint32 `db:"pk_id" orm:"pk,ai"`
	Username   string `db:"username"`
	Traffic    uint32 `db:"traffic"`
	HookCalls  uint32 `db:"hookCalls"`
	ResetIndex uint16 `db:"resetIndex"`
	CreatedAt  string `db:"createdAt"`
	IsValid    bool   `db:"isValid"`
	Role       Role   `db:"role"`
}

func getUserBySession(db *dbhelper.DBhelper, token string) (*User, error) {
	var user User
	err := db.QueryRowf(&user, `SELECT %s.pk_id, username, createdAt, isValid, traffic, hookCalls, role.pk_id "role.pk_id", role.name "role.name", role.maxPrivSources "role.maxPrivSources",role.maxPubSources "role.maxPubSources", role.maxSubscriptions "role.maxSubscriptions", role.maxHookCalls "role.maxHookCalls", role.maxTraffic "role.maxTraffic" FROM %s JOIN %s AS role ON (role.pk_id = %s.role) WHERE %s.pk_id=(SELECT userID FROM %s WHERE sessionToken=? AND isValid=1) and %s.isValid=1 LIMIT 1`,
		[]string{TableUser, TableUser, TableRoles, TableUser, TableUser, TableLoginSession, TableUser}, token)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func getUserByPK(db *dbhelper.DBhelper, pkID uint32) (*User, error) {
	var user User
	err := db.QueryRowf(&user, `SELECT %s.pk_id, username, traffic, hookCalls, createdAt, isValid, role.pk_id "role.pk_id", role.name "role.name", role.maxPrivSources "role.maxPrivSources", role.maxPubSources "role.maxPubSources",role.maxSubscriptions "role.maxSubscriptions", role.maxHookCalls "role.maxHookCalls", role.maxTraffic "role.maxTraffic" FROM %s JOIN %s AS role ON (role.pk_id = %s.role) WHERE %s.pk_id=? and %s.isValid=1 LIMIT 1`,
		[]string{TableUser, TableUser, TableRoles, TableUser, TableUser, TableUser}, pkID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func userExitst(db *dbhelper.DBhelper, username string) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE username=?", []string{TableUser}, username)
	return c > 0, err
}

func insertUser(db *dbhelper.DBhelper, username, password, ip string) error {
	_, err := db.Execf("INSERT INTO %s (username, password, ip) VALUES(?,?,?)", []string{TableUser}, username, password, ip)
	return err
}

//Returns count of affected rows
func resetUserResourceUsage(db *dbhelper.DBhelper) (int64, error) {
	rs, err := db.Execf("UPDATE %s SET resetIndex=+TIMESTAMPDIFF(MONTH, createdAt, now()), traffic=0, hookCalls=0 WHERE TIMESTAMPDIFF(MONTH, createdAt, now()) > resetIndex", []string{TableUser})
	if err != nil {
		return 0, err
	}
	return rs.RowsAffected()
}

func (user *User) isSubscribedTo(db *dbhelper.DBhelper, sourceID uint32) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE subscriber=? AND source=?", []string{TableSubscriptions}, user.Pkid, sourceID)
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

func (user *User) updateIP(db *dbhelper.DBhelper, ip string) error {
	return updateIP(db, user.Pkid, ip)
}

func (user *User) addHookCall(db *dbhelper.DBhelper, addTraffic uint32) error {
	_, err := db.Execf("UPDATE %s SET traffic=traffic+?, hookCalls=hookCalls+1 WHERE pk_id=?", []string{TableUser}, addTraffic, user.Pkid)
	return err
}
