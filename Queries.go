package main

import (
	"log"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Tables
const (
	TableUser          = "User"
	TableSources       = "Sources"
	TableLoginSession  = "LoginSessions"
	TableSubscriptions = "Subscriptions"
	TableModes         = "Modes"
	TableWebhooks      = "Webhooks"
)

func getInitSQL() dbhelper.QueryChain {
	return dbhelper.QueryChain{
		Name:  "initChain",
		Order: 0,
		Queries: dbhelper.CreateInitVersionSQL(
			//User
			dbhelper.InitSQL{
				//Create table
				Query:   "CREATE TABLE `%s` ( `pk_id` INT UNSIGNED NOT NULL AUTO_INCREMENT , `username` TEXT NOT NULL , `password` TEXT NOT NULL , `createdAt` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP , `isValid` BOOLEAN NOT NULL DEFAULT TRUE , PRIMARY KEY (`pk_id`)) ENGINE = InnoDB;",
				FParams: []string{TableUser},
			},

			//Sources
			dbhelper.InitSQL{
				//Create table
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `sourceID` text NOT NULL, `creator` int(10) unsigned NOT NULL, `mode` TINYINT UNSIGNED NOT NULL,`name` text NOT NULL, `description` text NOT NULL, `secret` varchar(48) NOT NULL, `creationTime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, `private` tinyint(1) NOT NULL DEFAULT '0', PRIMARY KEY (`pk_id`), KEY `creator` (`creator`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{TableSources},
			},
			dbhelper.InitSQL{
				//Create foreign key
				Query:   "ALTER TABLE `%s` ADD CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`creator`) REFERENCES `%s` (`pk_id`);",
				FParams: []string{TableSources, TableSources, TableUser},
			},

			//Modes
			dbhelper.InitSQL{
				Query:   "CREATE TABLE `%s` ( `modeID` TINYINT UNSIGNED NOT NULL, `name` text NOT NULL, PRIMARY KEY (`modeID`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{TableModes},
			},
			dbhelper.InitSQL{
				Query: "INSERT INTO `Modes` (`modeID`, `name`) VALUES ('0', 'Custom'), ('1', 'Gitlab'), ('2', 'Docker'), ('3', 'Github')",
			},
			dbhelper.InitSQL{
				//Create foreign key sources.mode -> modes.modeID
				Query:   "ALTER TABLE `%s` ADD CONSTRAINT `%s_ibfk_2` FOREIGN KEY (`mode`) REFERENCES `%s` (`modeID`);",
				FParams: []string{TableSources, TableSources, TableModes},
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

			//Subscriptions
			dbhelper.InitSQL{
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `subscriptionID` text NOT NULL, `subscriber` int(10) unsigned NOT NULL, `source` int(10) unsigned NOT NULL, `callbackURL` text, `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, `isValid` tinyint(1) NOT NULL DEFAULT '0', `lastTrigger` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00', PRIMARY KEY (`pk_id`), KEY `subscriber` (`subscriber`), KEY `source` (`source`), CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`subscriber`) REFERENCES `%s` (`pk_id`), CONSTRAINT `%s_ibfk_2` FOREIGN KEY (`source`) REFERENCES `%s` (`pk_id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{TableSubscriptions, TableSubscriptions, TableUser, TableSubscriptions, TableSources},
			},

			//Insert user
			dbhelper.InitSQL{
				Query:   "INSERT INTO `%s` (`pk_id`, `username`, `password`) VALUES (1, 'nouser', '');",
				FParams: []string{TableUser},
			},

			//Webhooks
			dbhelper.InitSQL{
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `sourceID` int(10) unsigned NOT NULL, `header` text NOT NULL, `payload` text NOT NULL, `received` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (`pk_id`), KEY `fkeySource` (`sourceID`), CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`sourceID`) REFERENCES `%s` (`pk_id`)) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4",
				FParams: []string{TableWebhooks, TableWebhooks, TableSources},
			},
		),
	}
}

// -------------------- Database QUERIES ----

// ------> Selects

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

func checkSubscriptionExitsts(db *dbhelper.DBhelper, sourceID uint32, callbackURL string) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE source=? AND callbackURL=?", []string{TableSubscriptions}, sourceID, callbackURL)
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

func (user *User) isSubscribedTo(db *dbhelper.DBhelper, sourceID uint32) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE subscriber=? AND source=?", []string{TableSubscriptions}, user.Pkid, sourceID)
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

func getSourceFromSourceID(db *dbhelper.DBhelper, sourceID string) (*Source, error) {
	var source Source
	err := db.QueryRowf(&source, "SELECT * FROM %s WHERE sourceID=?", []string{TableSources}, sourceID)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func getSourcesForUser(db *dbhelper.DBhelper, userID uint32) ([]Source, error) {
	var sources []Source
	err := db.QueryRowsf(&sources, "SELECT * FROM %s WHERE creator=?", []string{TableSources}, userID)
	if err != nil {
		return nil, err
	}
	return sources, nil
}

func getSubscriptionsForSource(db *dbhelper.DBhelper, sourceID uint32) ([]Subscription, error) {
	var subscriptions []Subscription
	err := db.QueryRowsf(&subscriptions, "SELECT * FROM %s WHERE source=?", []string{TableSubscriptions}, sourceID)
	return subscriptions, err
}

func getSubscriptionFromPK(db *dbhelper.DBhelper, pkID uint32) (*Subscription, error) {
	var subscription Subscription
	err := db.QueryRowf(&subscription, "SELECT * FROM %s WHERE pk_id=?", []string{TableSubscriptions}, pkID)
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

func getSourceFromPK(db *dbhelper.DBhelper, sourceID uint32) (*Source, error) {
	var source Source
	err := db.QueryRowf(&source, "SELECT * FROM %s WHERE pk_id=?", []string{TableSources}, sourceID)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func getWebhookFromPK(db *dbhelper.DBhelper, webhookID uint32) (*Webhook, error) {
	var webhook Webhook
	err := db.QueryRowf(&webhook, "SELECT * FROM %s WHERE pk_id=?", []string{TableWebhooks}, webhookID)
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (user *User) hasSourceWithName(db *dbhelper.DBhelper, name string) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE name=? AND creator=?", []string{TableSources}, name, user.Pkid)
	return c > 0, err
}

//Delete webhooks which aren't used anymore
func deleteOldHooks(db *dbhelper.DBhelper) {
	_, err := db.Execf("DELETE FROM %s WHERE (%s.received < (SELECT MIN(lastTrigger) FROM %s WHERE %s.source = %s.sourceID) AND DATE_ADD(received, INTERVAL 1 day) <= now()) OR DATE_ADD(received, INTERVAL 2 day) <= now()", []string{TableWebhooks, TableWebhooks, TableSubscriptions, TableSubscriptions, TableWebhooks})
	if err != nil {
		log.Println(err.Error())
	}
	log.Println("Webhook cleanup done")
}

// ----------> Updates

func (sub *Subscription) triggerAndValidate(db *dbhelper.DBhelper) error {
	_, err := db.Execf("UPDATE %s SET isValid=1, lastTrigger=now() WHERE subscriptionID=?", []string{TableSubscriptions}, sub.SubscriptionID)
	return err
}

func (sub *Subscription) trigger(db *dbhelper.DBhelper) {
	db.Execf("UPDATE %s SET lastTrigger=now() WHERE pk_id=?", []string{TableSubscriptions}, sub.PkID)
}

// -----------> Inserts

func (sub *Subscription) insert(db *dbhelper.DBhelper) error {
	sub.SubscriptionID = gaw.RandString(32)
	rs, err := db.Execf("INSERT INTO %s (subscriptionID, subscriber, source, callbackURL) VALUES (?,?,?,?)", []string{TableSubscriptions}, sub.SubscriptionID, sub.UserID, sub.Source, sub.CallbackURL)
	if err != nil {
		return err
	}
	id, err := rs.LastInsertId()
	if err != nil {
		return err
	}
	sub.PkID = uint32(id)
	return nil
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
	rs, err := db.Execf("INSERT INTO %s (creator, name, description, secret, private, sourceID, mode) VALUES(?,?,?,?,?,?,?)", []string{TableSources}, source.Creator.Pkid, source.Name, source.Description, secret, source.IsPrivate, sid, source.Mode)
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

func (webhook *Webhook) insert(db *dbhelper.DBhelper) error {
	rs, err := db.Execf("INSERT INTO %s (sourceID, header, payload) VALUES(?,?,?)", []string{TableWebhooks}, webhook.SourceID, webhook.Headers, webhook.Payload)
	if err != nil {
		return err
	}
	id, err := rs.LastInsertId()
	if err != nil {
		return err
	}
	webhook.PkID = uint32(id)
	return nil
}

// ----------> Deletes

//Delete source
func (source *Source) delete(db *dbhelper.DBhelper) error {
	_, err := db.Execf("DELETE FROM %s WHERE sourceID=?", []string{TableWebhooks}, source.PkID)
	if err != nil {
		return err
	}
	_, err = db.Execf("DELETE FROM %s WHERE source=?", []string{TableSubscriptions}, source.PkID)
	if err != nil {
		return err
	}
	_, err = db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSources}, source.PkID)
	return err
}

func removeSubscriptionByPK(db *dbhelper.DBhelper, pk uint32) error {
	_, err := db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSubscriptions}, pk)
	return err
}

func removeSubscription(db *dbhelper.DBhelper, subscriptionID string) error {
	_, err := db.Execf("DELETE FROM %s WHERE subscriptionID=?", []string{TableSubscriptions}, subscriptionID)
	return err
}
