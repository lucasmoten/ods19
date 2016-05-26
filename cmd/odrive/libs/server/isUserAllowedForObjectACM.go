package server

import (
	"errors"
	"log"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/performance"
)

func (h AppServer) isUserAllowedForObjectACM(ctx context.Context, object *models.ODObject) (bool, error) {
	var err error
	// In standalone, we are ignoring AAC
	if config.StandaloneMode {
		// But warn in STDOUT to draw attention
		log.Printf("WARNING: STANDALONE mode is active. User permission to access objects are not being checked against AAC.")
		// Return permission granted and no errors
		return true, nil
	}

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

	// Verify we have a reference to AAC
	if h.AAC == nil {
		return false, errors.New("AAC field is nil")
	}

	// Performance instrumentation
	var beganAt = performance.BeganJob(int64(0))
	if h.Tracker != nil {
		beganAt = h.Tracker.BeginTime(performance.AACCounterCheckAccess)
	}

	// Call AAC
	aacResponse, err := h.AAC.CheckAccess(dn, tokenType, acm)

	// End performance tracking for the AAC call
	if h.Tracker != nil {
		h.Tracker.EndTime(
			performance.AACCounterCheckAccess,
			beganAt,
			performance.SizeJob(1),
		)
	}

	// Check if there was an error calling the service
	if err != nil {
		log.Printf("Error calling AAC.CheckAccess: %s", err.Error())
		return false, errors.New("Error calling AAC.CheckAccess")
	}

	// Process Response
	// Log the messages
	for _, message := range aacResponse.Messages {
		log.Printf("Message in AAC Response: %s\n", message)
	}
	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !aacResponse.Success {
		return false, errors.New("Response from AAC.CheckAccess failed")
	}
	// AAC Response returned without error, was successful
	return aacResponse.HasAccess, nil
}
