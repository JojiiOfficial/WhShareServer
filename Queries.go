package main

import (
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

func getInitSQL() dbhelper.QueryChain {
	return dbhelper.QueryChain{
		Name:  "initChain",
		Order: 0,
		Queries: dbhelper.CreateInitVersionSQL(
			dbhelper.InitSQL{
				Query: "CREATE TABLE Sources (pk_id int(10) UNSIGNED NOT NULL, creator int(10) UNSIGNED NOT NULL, name text NOT NULL, secret varchar(48) NOT NULL, creationTime int(11) NOT NULL, private tinyint(1) NOT NULL DEFAULT '0' ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			},
			dbhelper.InitSQL{
				Query: "ALTER TABLE Sources ADD PRIMARY KEY (pk_id)",
			},
			dbhelper.InitSQL{
				Query: "ALTER TABLE Sources MODIFY pk_id int(10) UNSIGNED NOT NULL AUTO_INCREMENT",
			},
		),
	}
}
