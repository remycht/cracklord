package main

import (
	"crypto/rand"
	log "github.com/Sirupsen/logrus"
	"github.com/jmckaskill/gokerb"
	"github.com/jmmcatee/goldap/ad"
	"strings"
	"time"
)

// Active Directory structure to implement the basic authenticator
type ADAuth struct {
	GroupMap map[string]string
	realm    string
}

// Function to configure the group mappying. One AD group per server group
// constant is expected.
func (a *ADAuth) Setup(mapping map[string]string) {
	a.GroupMap = mapping
	log.Debug("AD authentication setup complete")
}

// Function to configure the realm of the AD auth
func (a *ADAuth) SetRealm(realm string) {
	a.realm = strings.ToUpper(realm)
	log.WithField("realm", realm).Debug("AD authentication realm set.")
}

// Function to log in a user
func (a *ADAuth) Login(user, pass string) (User, error) {
	// Setup Credential Config
	credConf := kerb.CredConfig{
		Dial: kerb.DefaultDial,
		Now:  time.Now,
		Rand: rand.Reader,
	}

	logger := log.WithFields(log.Fields{
		"user":  user,
		"realm": a.realm,
	})

	// Verify the validity of user and password
	creds, err := kerb.NewCredential(user, a.realm, pass, &credConf)
	if err != nil {
		logger.Error("Error verifying kerberos credentials.")
		return User{}, err
	}
	logger.Debug("Validated kerberos credentials.")	

	// Get a ticket to prove the creds are valid
	_, err = creds.GetTicket("krbtgt/"+a.realm, nil)
	if err != nil {
		logger.WithField("error", err.Error()).Error("Error gathering kerberos ticket.")
		return User{}, err
	}

	// User is valid so get group membership
	db := ad.New(creds, a.realm)

	// Get the user info from AD
	logger.Debug("Attempting to enumerate LDAP user info from AD")
	adUser, err := db.LookupPrincipal(user, a.realm)

	NewUser := User{
		Username: user,
	}

	for _, g := range adUser.Member {
		logger.WithField("group", g).Debug("Checking AD group.")
		// Check if the AD group has a mapping
		if clGroup, ok := a.GroupMap[g.String()]; ok {
			logger.WithField("group", clGroup).Debug("Adding group to list")
			// Group existed so store the result in the User structure
			NewUser.Groups = append(NewUser.Groups, clGroup)
		}
	}

	// User is logged in now
	NewUser.LogOnTime = time.Now()

	// Expiration timer is handled by the TokenStore

	return NewUser, nil
}
