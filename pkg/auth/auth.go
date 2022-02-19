package auth

import (
	"fmt"
	"os"
)

// Authenticator provides an interface to authenticate
// privileged user who are the only ones being able to
// access certain api methods.
type Authenticator interface {

	// CheckAuthentication checks whether the given username
	// and password are correct. True will be returned, if it
	// is the case. Otherwise, false.
	CheckAuthentication(username, password string) bool
}

// Credentials is an object containing a  username
// and corresponding password in plain text.
type Credentials struct {
	username string
	password string
}

// CachedCredentials is an Authenticator that stores the
// username and password in plain text in cache.
type CachedCredentials struct {
	credentials Credentials
}

// NewEnvironmentBasedAuthentication expects a single username
// and password specified as environment variables. The username
// must be specified as 'BLU_AUTH_USERNAME', and the password must
// be specified as 'BLU_AUTH_PASSWORD'. If those two values are
// specified, then a valid Authenticator will be returned.
//
// Otherwise, if only one of them isn't specified, an error will
// be returned.
func NewEnvironmentBasedAuthentication() (Authenticator, error) {
	username := os.Getenv("BLU_AUTH_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("no username specified in the environment (BLU_AUTH_USERNAME)")
	}
	password := os.Getenv("BLU_AUTH_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("no password specified in the environment (BLU_AUTH_PASSWORD)")
	}
	return &CachedCredentials{credentials: Credentials{username: username, password: password}}, nil
}

func (auth *CachedCredentials) CheckAuthentication(username, password string) bool {
	return auth.credentials.username == username && auth.credentials.password == password
}
