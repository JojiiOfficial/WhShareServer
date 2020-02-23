package models

import (
	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

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

//TableUser the table in db for user
const TableUser = "User"

//HasSourceWithName if user has source with name
func (user *User) HasSourceWithName(db *dbhelper.DBhelper, name string) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE name=? AND creator=?", []string{TableSources}, name, user.Pkid)
	return c > 0, err
}

//GetSourceCount gets the count of sources for an user
func (user *User) GetSourceCount(db *dbhelper.DBhelper, private bool) (uint, error) {
	var c uint
	priv := 0
	if private {
		priv = 1
	}
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE creator=? AND private=?", []string{TableSources}, user.Pkid, priv)
	return c, err
}

//GetUserBySession get user by sessionToken
func GetUserBySession(db *dbhelper.DBhelper, token string) (*User, error) {
	var user User
	err := db.QueryRowf(&user, `SELECT %s.pk_id, username, createdAt, isValid, traffic, hookCalls, role.pk_id "role.pk_id", role.name "role.name", role.maxPrivSources "role.maxPrivSources",role.maxPubSources "role.maxPubSources", role.maxSubscriptions "role.maxSubscriptions", role.maxHookCalls "role.maxHookCalls", role.maxTraffic "role.maxTraffic" FROM %s JOIN %s AS role ON (role.pk_id = %s.role) WHERE %s.pk_id=(SELECT userID FROM %s WHERE sessionToken=? AND isValid=1) and %s.isValid=1 LIMIT 1`,
		[]string{TableUser, TableUser, TableRoles, TableUser, TableUser, TableLoginSession, TableUser}, token)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

//GetUserByPK get user by pk_id
func GetUserByPK(db *dbhelper.DBhelper, pkID uint32) (*User, error) {
	var user User
	err := db.QueryRowf(&user, `SELECT %s.pk_id, username, traffic, hookCalls, createdAt, isValid, role.pk_id "role.pk_id", role.name "role.name", role.maxPrivSources "role.maxPrivSources", role.maxPubSources "role.maxPubSources",role.maxSubscriptions "role.maxSubscriptions", role.maxHookCalls "role.maxHookCalls", role.maxTraffic "role.maxTraffic" FROM %s JOIN %s AS role ON (role.pk_id = %s.role) WHERE %s.pk_id=? and %s.isValid=1 LIMIT 1`,
		[]string{TableUser, TableUser, TableRoles, TableUser, TableUser, TableUser}, pkID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

//UserExists if user exists
func UserExists(db *dbhelper.DBhelper, username string) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE username=?", []string{TableUser}, username)
	return c > 0, err
}

//InsertUser inserts user into db
func InsertUser(db *dbhelper.DBhelper, username, password, ip string) error {
	_, err := db.Execf("INSERT INTO %s (username, password, ip) VALUES(?,?,?)", []string{TableUser}, username, password, ip)
	return err
}

//IsSubscribedTo if user subscribed the given source
func (user *User) IsSubscribedTo(db *dbhelper.DBhelper, sourceID uint32) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE subscriber=? AND source=?", []string{TableSubscriptions}, user.Pkid, sourceID)
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

//UpdateIP for an user
func (user *User) UpdateIP(db *dbhelper.DBhelper, ip string) error {
	return updateIP(db, user.Pkid, ip)
}

//AddHookCall adds hookCall (increase hookCall count and traffic)
func (user *User) AddHookCall(db *dbhelper.DBhelper, addTraffic uint32) error {
	_, err := db.Execf("UPDATE %s SET traffic=traffic+?, hookCalls=hookCalls+1 WHERE pk_id=?", []string{TableUser}, addTraffic, user.Pkid)
	return err
}

//GetSubscriptionCount gets count of subscriptions for an user
func (user *User) GetSubscriptionCount(db *dbhelper.DBhelper) (uint32, error) {
	var c uint32
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE subscriber=?", []string{TableSubscriptions}, user.Pkid)
	return c, err
}

//LoginQuery loginQuery
func LoginQuery(db *dbhelper.DBhelper, username, password, ip string) (string, bool, error) {
	var pkid uint32
	err := db.WithHook(dbhelper.NoHook).QueryRowf(&pkid, "SELECT pk_id FROM %s WHERE username=? AND password=? AND isValid=1 LIMIT 1", []string{TableUser}, username, password)
	if err != nil || pkid < 1 {
		return "", false, nil
	}

	session := LoginSession{
		UserID: pkid,
		Token:  gaw.RandString(64),
	}

	err = session.Insert(db)
	if LogError(err) {
		return "", false, err
	}

	updateIP(db, pkid, ip)

	return session.Token, true, nil
}

func updateIP(db *dbhelper.DBhelper, userID uint32, ip string) error {
	_, err := db.Execf("UPDATE %s SET ip=? WHERE pk_id=?", []string{TableUser}, ip, userID)
	return err
}
