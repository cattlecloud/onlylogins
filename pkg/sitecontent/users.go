package sitecontent

import (
	"time"

	"cattlecloud.net/onlylogins/pkg/users"
)

type UserInfo struct {
	users.Profile
	Karma    int
	Standing users.Standing
}

type NameChange struct {
	Chosen    string
	Previous  string
	Timestamp time.Time
}

type UserProfile struct {
	UserInfo
	Company   string
	Location  string
	Link      string
	ShowEmail bool
	About     string
}
