package server

import (
	"errors"
	"log"

	"golang.org/x/net/context"

	"decipher.com/oduploader/metadata/models"
)

func (h AppServer) isUserAllowedForObjectACM(ctx context.Context, object *models.ODObject) (bool, error) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return false, errors.New("Could not determine user")

	}

	// Validate object
	if object == nil {
		return false, errors.New("Object passed in is not initialized")
	}
	if !object.RawAcm.Valid {
		return false, errors.New("Object passed in does not have an ACM set")
	}

	// Gather inputs
	tokenType := "pki_dias"
	dn := caller.DistinguishedName
	acm := object.RawAcm.String

	// Debug of what is being passed
	//log.Println(fmt.Printf("Calling AAC.CheckAccess(dn='%s', tokenType='%s', acm='%s')", dn, tokenType, acm))

	// Call AAC
	aacResponse, err := h.AAC.CheckAccess(dn, tokenType, acm)

	// Process Response
	if err != nil {
		return false, errors.New("Error calling ACM")
	}
	// Log the messages
	for _, message := range aacResponse.Messages {
		log.Printf("Message in AAC Response: %s\n", message)
	}
	if !aacResponse.Success {
		return false, errors.New("Response from ACM failed")
	}

	// AAC Response returned without error, was successful
	return aacResponse.HasAccess, nil
}
