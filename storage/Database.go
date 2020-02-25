package storage

import (
	"strconv"

	log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
)

//ConnectDB connects to MySQL
func ConnectDB(config *models.ConfigStruct, isDebug, nocolor bool) (*dbhelper.DBhelper, error) {
	log.Debug("Connecting to DB")
	db, err := dbhelper.NewDBHelper(dbhelper.Mysql).Open(
		config.Server.Database.Username,
		config.Server.Database.Pass,
		config.Server.Database.Host,
		strconv.Itoa(config.Server.Database.DatabasePort),
		config.Server.Database.Database,
		"parseTime=True",
	)

	if err != nil {
		return nil, err
	}

	log.Info("Connected successfully")

	//Only debugMode if logLevel is debug
	db.Options.Debug = isDebug
	db.Options.UseColors = !nocolor

	return db, updateDB(db)
}

func updateDB(db *dbhelper.DBhelper) error {
	db.AddQueryChain(getInitSQL())
	return db.RunUpdate()
}

func getInitSQL() dbhelper.QueryChain {
	return dbhelper.QueryChain{
		Name:  "initChain",
		Order: 0,
		Queries: dbhelper.CreateInitVersionSQL(
			//Roles
			dbhelper.InitSQL{
				//Create role table
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `name` text NOT NULL, `maxPubSources` int(11) NOT NULL, `maxPrivSources` int(11) NOT NULL, `maxSubscriptions` int(11) NOT NULL, `maxHookCalls` int(11) NOT NULL COMMENT 'per month', `maxTraffic` int(11) NOT NULL COMMENT 'in kb', PRIMARY KEY (`pk_id`)) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4",
				FParams: []string{models.TableRoles},
			},
			dbhelper.InitSQL{
				//Insert default roles
				Query:   "INSERT INTO `%s` (`pk_id`, `name`, `maxPrivSources`,`maxPubSources`, `maxSubscriptions`, `maxHookCalls`, `maxTraffic`) VALUES (1, 'guest',0, 0, -1, 0, 0), (2, 'admin', -1, -1 ,-1, -1, -1), (3, 'user', 2, 40, 100, 50, 10000);",
				FParams: []string{models.TableRoles},
			},

			//User
			dbhelper.InitSQL{
				//Create table
				Query:   "CREATE TABLE `%s` ( `pk_id` INT UNSIGNED NOT NULL AUTO_INCREMENT , `username` TEXT NOT NULL , `password` TEXT NOT NULL , `ip` varchar(16) NOT NULL, `role` int(10) unsigned NOT NULL, `traffic` int(10) unsigned NOT NULL COMMENT 'in bytes', `hookCalls` int(10) unsigned NOT NULL, `resetIndex` smallint(5) unsigned NOT NULL, `createdAt` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP , `isValid` BOOLEAN NOT NULL DEFAULT TRUE , PRIMARY KEY (`pk_id`), KEY `role` (`role`), CONSTRAINT `User_ibfk_1` FOREIGN KEY (`role`) REFERENCES `Roles` (`pk_id`)) ENGINE = InnoDB;",
				FParams: []string{models.TableUser},
			},

			//Sources
			dbhelper.InitSQL{
				//Create table
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `sourceID` text NOT NULL, `creator` int(10) unsigned NOT NULL, `mode` TINYINT UNSIGNED NOT NULL,`name` text NOT NULL, `description` text NOT NULL, `secret` varchar(48) NOT NULL, `creationTime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, `private` tinyint(1) NOT NULL DEFAULT '0', PRIMARY KEY (`pk_id`), KEY `creator` (`creator`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{models.TableSources},
			},
			dbhelper.InitSQL{
				//Create foreign key
				Query:   "ALTER TABLE `%s` ADD CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`creator`) REFERENCES `%s` (`pk_id`);",
				FParams: []string{models.TableSources, models.TableSources, models.TableUser},
			},

			//Modes
			dbhelper.InitSQL{
				Query:   "CREATE TABLE `%s` ( `modeID` TINYINT UNSIGNED NOT NULL, `name` text NOT NULL, PRIMARY KEY (`modeID`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{models.TableModes},
			},
			dbhelper.InitSQL{
				Query: "INSERT INTO `Modes` (`modeID`, `name`) VALUES ('0', 'Custom'), ('1', 'Gitlab'), ('2', 'Docker'), ('3', 'Github')",
			},
			dbhelper.InitSQL{
				//Create foreign key sources.mode -> modes.modeID
				Query:   "ALTER TABLE `%s` ADD CONSTRAINT `%s_ibfk_2` FOREIGN KEY (`mode`) REFERENCES `%s` (`modeID`);",
				FParams: []string{models.TableSources, models.TableSources, models.TableModes},
			},

			//LoginSessions
			dbhelper.InitSQL{
				//Create table
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `userID` int(10) unsigned NOT NULL, `sessionToken` varchar(64) NOT NULL, `created` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, `isValid` tinyint(1) NOT NULL DEFAULT '1', PRIMARY KEY (`pk_id`), KEY `userID` (`userID`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{models.TableLoginSession},
			},
			dbhelper.InitSQL{
				//Create foreign key
				Query:   "ALTER TABLE `%s` ADD CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`userID`) REFERENCES `%s` (`pk_id`);",
				FParams: []string{models.TableLoginSession, models.TableLoginSession, models.TableUser},
			},

			//Subscriptions
			dbhelper.InitSQL{
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `subscriptionID` text NOT NULL, `subscriber` int(10) unsigned NOT NULL, `source` int(10) unsigned NOT NULL, `callbackURL` text, `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, `isValid` tinyint(1) NOT NULL DEFAULT '0', `lastTrigger` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00', PRIMARY KEY (`pk_id`), KEY `subscriber` (`subscriber`), KEY `source` (`source`), CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`subscriber`) REFERENCES `%s` (`pk_id`), CONSTRAINT `%s_ibfk_2` FOREIGN KEY (`source`) REFERENCES `%s` (`pk_id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				FParams: []string{models.TableSubscriptions, models.TableSubscriptions, models.TableUser, models.TableSubscriptions, models.TableSources},
			},

			//Insert user
			dbhelper.InitSQL{
				Query:   "INSERT INTO `%s` (`pk_id`, `username`, `password`, `ip`, `role`) VALUES (1, 'nouser', '','-','1');",
				FParams: []string{models.TableUser},
			},

			//Webhooks
			dbhelper.InitSQL{
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT, `sourceID` int(10) unsigned NOT NULL, `header` text NOT NULL, `payload` text NOT NULL, `received` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (`pk_id`), KEY `fkeySource` (`sourceID`), CONSTRAINT `%s_ibfk_1` FOREIGN KEY (`sourceID`) REFERENCES `%s` (`pk_id`)) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4",
				FParams: []string{models.TableWebhooks, models.TableWebhooks, models.TableSources},
			},
		),
	}
}
