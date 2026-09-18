package users

import (
	"math/rand/v2"
	"strings"
	"time"

	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/users/oauths"
)

const (
	// Inactive is a sentinel value representing no user.
	Inactive = -1
)

// Identitiy is the unique database id assigned to each user profile.
//
// It is just an auto increment, but that is an implementation detail.
type Identity records.Auto

type Profile struct {
	ID       Identity
	Username string
	Email    string
	IDP      oauths.IDP
}

func (up *Profile) Active() bool {
	return up != nil
}

func (up *Profile) Identity() identity.UserIdentity {
	if up == nil {
		return Inactive
	}
	return up.ID
}

func InitialProfile(idp oauths.IDP, sub oauths.Subject, email string) *CreateUserRequest {
	now := time.Now()
	u := GenerateUsername(now)
	return &CreateUserRequest{
		IDP:      idp,
		Subject:  sub,
		Email:    email,
		Username: u,
	}
}

func GenerateUsername(now time.Time) string {
	sb := new(strings.Builder)
	sb.WriteString("user_")
	b32 := format.Base32(now.Unix())
	sb.WriteString(b32)
	suffix1 := byte(rand.IntN(26) + 97)
	suffix2 := byte(rand.IntN(26) + 97)
	sb.Write([]byte{suffix1, suffix2})
	return sb.String()
}

type CreateUserRequest struct {
	Username string
	Email    string
	Subject  oauths.Subject
	IDP      oauths.IDP
}
