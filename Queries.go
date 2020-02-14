package main

import (
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
				Query:   "CREATE TABLE `%s` (`pk_id` int(10) unsigned NOT NULL AUTO_INCREMENT,`creator` int(10) unsigned NOT NULL, `name` text NOT NULL, `secret` varchar(48) NOT NULL, `creationTime` int(11) NOT NULL, `private` tinyint(1) NOT NULL DEFAULT '0', PRIMARY KEY (`pk_id`), KEY `creator` (`creator`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
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
