package oauths

// Subject is the JWT sub field of an authentication token.
type Subject string

// IDP is the enum associated with the oauths database table, and must
// be kept in sync.
type IDP int

const (
	Testing   IDP = 1
	Google    IDP = 2
	Apple     IDP = 3
	Microsoft IDP = 4
	GitHub    IDP = 5
)
