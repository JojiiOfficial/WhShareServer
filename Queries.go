package main

import (
	"log"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Tables
const (
	TableUser         = "User"
	TableSources      = "Sources"
	TableLoginSession = "LoginSessions"
)

func getInitSQL() dbhelper.QueryChain {
	return dbhelper.QueryChain{
		Name:  "initChain",
		Order: 0,
		Queries: dbhelper.CreateInitVersionSQL(
			//User
			dbhelper.InitSQL{
				//Create table
				Query:   "CREATE TABLE `WeShare`.`%s` ( `pk_id` INT UNSIGNED NOT NULL AUTO_INCREMENT , `username` TEXT NOT NULL , `password` TEXT NOT NULL , `createdAt` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP , `isValid` BOOLEAN NOT NULL DEFAULT TRUE , PRIMARY KEY (`pk_id`)) ENGINE = InnoDB;",
				FParams: []string{TableUser},
			},

			//Sources
			dbhelper.InitSQL{
				//Create table
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT,`creator` int(10) unsigned NOT NULL, `name` text NOT NULL, `description` text NOT NULL, `secret` varchar(48) NOT NULL, `creationTime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, `private` tinyint(1) NOT NULL DEFAULT '0', PRIMARY KEY (`pk_id`), KEY `creator` (`creator`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{TableSources},
			},
			dbhelper.InitSQL{
				//Create foreign key
				Query:   "ALTER TABLE `%s` ADD CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`creator`) REFERENCES `%s` (`pk_id`);",
				FParams: []string{TableSources, TableSources, TableUser},
			},

			//LoginSessions
			dbhelper.InitSQL{
				//Create table
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `userID` int(10) unsigned NOT NULL, `sessionToken` varchar(64) NOT NULL, `created` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, `isValid` tinyint(1) NOT NULL DEFAULT '1', PRIMARY KEY (`pk_id`), KEY `userID` (`userID`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{TableLoginSession},
			},
			dbhelper.InitSQL{
				//Create foreign key
				Query:   "ALTER TABLE `%s` ADD CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`userID`) REFERENCES `%s` (`pk_id`);",
				FParams: []string{TableLoginSession, TableLoginSession, TableUser},
			},
		),
	}
}

// -------------------- Database QUERIES ----

func loginQuery(db *dbhelper.DBhelper, username, password string) (string, bool, error) {
	var pkid uint32
	err := db.QueryRowf(&pkid, "SELECT pk_id FROM %s WHERE username=? AND password=? AND isValid=1", []string{TableUser}, username, password)
	if err != nil || pkid < 1 {
		return "", false, nil
	}

	session := LoginSession{
		UserID: pkid,
		Token:  gaw.RandString(64),
	}

	err = session.insert(db)
	if err != nil {
		log.Println(err.Error())
		return "", false, err
	}

	return session.Token, true, nil
}

func getUserIDFromSession(db *dbhelper.DBhelper, token string) (*User, error) {
	var user User
	err := db.QueryRowf(&user, "SELECT pk_id, username, createdAt, isValid FROM %s WHERE %s.pk_id=(SELECT userID FROM %s WHERE sessionToken=? AND isValid=1) and %s.isValid=1",
		[]string{TableUser, TableUser, TableLoginSession, TableUser}, token)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (session *LoginSession) insert(db *dbhelper.DBhelper) error {
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

func (source *Source) insert(db *dbhelper.DBhelper) error {
	secret := gaw.RandString(48)
	sid := gaw.RandString(32)
	rs, err := db.Execf("INSERT INTO %s (creator, name, description, secret, private, sourceID) VALUES(?,?,?,?,?,?)", []string{TableSources}, source.Creator.Pkid, source.Name, source.Description, secret, source.IsPrivate, sid)
	if err != nil {
		return err
	}
	id, err := rs.LastInsertId()
	if err != nil {
		return err
	}
	source.PkID = uint32(id)
	source.Secret = secret
	source.SourceID = sid

	return nil
}
