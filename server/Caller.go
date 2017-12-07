package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/deciphernow/object-drive-server/config"
	"github.com/deciphernow/object-drive-server/protocol"
)

// protocolCaller converts the server.Caller type to a protocol.Caller type. Recommending
// that server.Caller be deprecated.
func protocolCaller(caller Caller) protocol.Caller {
	return protocol.Caller{
		DistinguishedName:               caller.DistinguishedName,
		UserDistinguishedName:           caller.UserDistinguishedName,
		ExternalSystemDistinguishedName: caller.ExternalSystemDistinguishedName,
		SSLClientSDistinguishedName:     caller.SSLClientSDistinguishedName,
		CommonName:                      caller.CommonName,
		TransactionType:                 caller.TransactionType,
		Groups:                          caller.Groups,
	}
}

// Caller provides the distinguished names obtained from specific request
// headers and peer certificate if called directly
type Caller struct {
	// DistinguishedName is the unique identity of a user
	DistinguishedName string
	// UserDistinguishedName holds the value passed in header USER_DN
	UserDistinguishedName string
	// ExternalSystemDistinguishedName holds the value passed in header EXTERNAL_SYS_DN
	ExternalSystemDistinguishedName string
	// SSLClientSDistinguishedName holds the value passed in header SSL_CLIENT_S_DN
	SSLClientSDistinguishedName string
	// CommonName is the CN value part of the DistinguishedName
	CommonName string
	// TransactionType can be either NORMAL, IMPERSONATION, or UNKNOWN
	TransactionType string
	// Groups are extracted from the f_share fields for a Caller. Groups should be flattened
	// before comparing strings.
	Groups []string
}

// CallerFromRequest populates a Caller object based upon request headers and peer
// certificates. Logically this is intended to work with or without NGINX as
// a front end
func CallerFromRequest(r *http.Request) Caller {
	var caller Caller
	caller.UserDistinguishedName = config.GetNormalizedDistinguishedName(r.Header.Get("USER_DN"))
	caller.ExternalSystemDistinguishedName = r.Header.Get("EXTERNAL_SYS_DN")
	caller.SSLClientSDistinguishedName = r.Header.Get("SSL_CLIENT_S_DN")

	if isHTTPS(r) {
		// DO NOT NORMALIZE THE user because it comes from a certificate used directly
		if len(r.TLS.PeerCertificates) > 0 {
			caller.SSLClientSDistinguishedName = config.GetDistinguishedName(r.TLS.PeerCertificates[0])
		} else {
			caller.SSLClientSDistinguishedName = ""
		}
	}

	if caller.UserDistinguishedName != "" {
		caller.DistinguishedName = caller.UserDistinguishedName
	} else {
		caller.DistinguishedName = caller.SSLClientSDistinguishedName
	}
	caller.DistinguishedName = config.GetNormalizedDistinguishedName(caller.DistinguishedName)
	caller.CommonName = config.GetCommonName(caller.DistinguishedName)
	return caller
}

func isHTTPS(r *http.Request) bool {
	if r.TLS == nil || !r.TLS.HandshakeComplete {
		return false
	}
	return true
}

// ValidateHeaders examines the values picked up from the headers and determines if they are valid
func (c *Caller) ValidateHeaders(whitelist []string, r *http.Request) error {
	c.TransactionType = "IMPERSONATION"
	userDn := c.UserDistinguishedName
	sslClientSDn := c.SSLClientSDistinguishedName
	externalSysDn := c.ExternalSystemDistinguishedName

	if isHTTPS(r) {
		if !have(userDn) && have(sslClientSDn) && !have(externalSysDn) {
			// the ssl_client_s_dn is really the User and we don't need to look anything up.
			c.TransactionType = "NORMAL"
			return nil
		} else if have(userDn) && have(sslClientSDn) && have(externalSysDn) {
			c.TransactionType = "IMPERSONATION"
			if !canImpersonateUser(whitelist, sslClientSDn, userDn) ||
				!canImpersonateUser(whitelist, externalSysDn, userDn) {
				return fmt.Errorf("Unauthorized: Either or both of the ssl_client_s_dn or external_sys_dn are not authorized to impersonate or have access.")
			}
			return nil
		} else if have(userDn) && have(sslClientSDn) && !have(externalSysDn) {
			c.TransactionType = "IMPERSONATION"
			if !canImpersonateUser(whitelist, sslClientSDn, userDn) {
				return fmt.Errorf("Unauthorized: The ssl_client_s_dn is not authorized to impersonate or have access.")
			}
			return nil
		} else if !have(userDn) && have(sslClientSDn) && have(externalSysDn) {
			c.TransactionType = "IMPERSONATION"
			return fmt.Errorf("Unauthorized: Missing the user_dn")
		} else {
			c.TransactionType = "UNKNOWN"
			return fmt.Errorf("Unauthorized: Invalid connection. Required headers, user_dn and possibly external_sys_dn are missing.")
		}
	} else {
		c.TransactionType = "IMPERSONATION"
		if have(userDn) && have(sslClientSDn) && have(externalSysDn) {
			if !canImpersonateUser(whitelist, sslClientSDn, userDn) ||
				!canImpersonateUser(whitelist, externalSysDn, userDn) {
				return fmt.Errorf("Unauthorized: Either or both of the ssl_client_s_dn or external_sys_dn are not authorized to impersonate or have access.")
			}
			return nil
		} else if have(userDn) && have(sslClientSDn) {
			if !canImpersonateUser(whitelist, sslClientSDn, userDn) {
				return fmt.Errorf("Unauthorized: The ssl_client_s_dn is not authorized to impersonate or have access.")
			}
			return nil
		} else if have(userDn) {
			return fmt.Errorf("Unauthorized: Connection is not authorized to impersonate. Neither ssl_client_s_dn or external_sys_dn were found.")
		}
		return fmt.Errorf("Unauthorized: Invalid connection. Required headers are missing, user_dn, ssl_client_s_dn, and or external_sys_dn.")
	}
}

// GetMessage returns formatted state common for AclRestFilter logging
func (c *Caller) GetMessage() string {
	return " transport: https user_dn: " + c.UserDistinguishedName + " ssl_client_s_dn: " + c.SSLClientSDistinguishedName + " external_sys_dn: " + c.ExternalSystemDistinguishedName
}

func have(s string) bool {
	return len(s) > 0
}

func canImpersonateUser(whitelist []string, clientID string, user string) bool {

	normalizedClient := config.GetNormalizedDistinguishedName(clientID)
	normalizedUserToken := config.GetNormalizedDistinguishedName(user)

	if contains := whitelistContains(whitelist, clientID); !contains {
		log.Printf("Client %s is denied! Unable to impersonate %s", normalizedClient, normalizedUserToken)
		return false
	}
	return true
}

func whitelistContains(list []string, clientID string) bool {
	for _, v := range list {
		if strings.ToLower(config.GetNormalizedDistinguishedName(v)) == strings.ToLower(config.GetNormalizedDistinguishedName(clientID)) {
			return true
		}
	}
	return false
}
