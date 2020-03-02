package models

import (
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Role the role of a user
type Role struct {
	PkID             uint32 `db:"pk_id" orm:"pk,ai"`
	Name             string `db:"name"`
	MaxPrivSources   int    `db:"maxPrivSources"`
	MaxPubSources    int    `db:"maxPubSources"`
	MaxSubscriptions int    `db:"maxSubscriptions"`
	MaxHookCalls     int    `db:"maxHookCalls"`
	MaxTraffic       int    `db:"maxTraffic"`
	IsAdmin          bool   `db:"isAdmin"`
}

//TableRoles the db tableName for the roles
const TableRoles = "Roles"

//CanCreateSource returns true if a role allows having private/public a source
func (user User) CanCreateSource(private bool) bool {
	return !((private && user.Role.MaxPrivSources == 0) || (!private && user.Role.MaxPubSources == 0))
}

//CanShareWebhooks return true if role is allowed to send webhooks
func (user User) CanShareWebhooks() bool {
	return !(user.Role.MaxTraffic == 0 || user.Role.MaxHookCalls == 0)
}

//HasUnlimitedHookCalls return true if user has unlimited hook calls
func (user User) HasUnlimitedHookCalls() bool {
	return user.Role.MaxHookCalls == -1 && user.Role.MaxTraffic == -1
}

//IsSourceLimitReached return true if source limit is reached
func (user User) IsSourceLimitReached(db *dbhelper.DBhelper, private bool) (bool, error) {
	//Get count of sources
	scount, err := user.GetSourceCount(db, private)
	if err != nil {
		return false, err
	}

	//Check for source limit
	return (private && user.Role.MaxPrivSources != -1 && scount >= uint(user.Role.MaxPrivSources)) ||
		(!private && user.Role.MaxPubSources != -1 && scount >= uint(user.Role.MaxPubSources)), nil
}

//IsSubscriptionLimitReached return true if users subscription limit is reached
func (user User) IsSubscriptionLimitReached(db *dbhelper.DBhelper) (bool, error) {
	userSubscriptions, err := user.GetSubscriptionCount(db)
	if err != nil {
		return false, err
	}

	return user.Role.MaxSubscriptions != -1 && userSubscriptions >= uint32(user.Role.MaxSubscriptions), nil
}

//CanSubscribe return true if user can subscribe to a source
func (user User) CanSubscribe() bool {
	return user.Role.MaxSubscriptions != 0
}
