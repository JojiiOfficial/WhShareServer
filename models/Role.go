package models

//Role the role of a user
type Role struct {
	PkID             uint32 `db:"pk_id" orm:"pk,ai"`
	Name             string `db:"name"`
	MaxPrivSources   int    `db:"maxPrivSources"`
	MaxPubSources    int    `db:"maxPubSources"`
	MaxSubscriptions int    `db:"maxSubscriptions"`
	MaxHookCalls     int    `db:"maxHookCalls"`
	MaxTraffic       int    `db:"maxTraffic"`
}

//TableRoles the db tableName for the roles
const TableRoles = "Roles"
