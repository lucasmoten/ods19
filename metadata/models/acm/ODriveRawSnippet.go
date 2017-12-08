package acm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/config"
	"github.com/deciphernow/object-drive-server/utils"
)

var (
	logger = config.RootLogger
)

// ODriveRawSnippet is a structure to hold the snippets returned from an AAC GetSnippets response where snippetType = 'odrive-raw'
type ODriveRawSnippet struct {
	FlatMACs               string `json:"f_macs"`
	FlatOCOrgs             string `json:"f_oc_org"`
	FlatSAP                string `json:"f_sap"`
	FlatClearance          string `json:"f_clearance"`
	FlatAtomEnergy         string `json:"f_aea"`
	FlatSCIControls        string `json:"f_sci_ctrls"`
	DisseminationCountries string `json:"dissem_countries"`
	FlatACCMs              string `json:"f_accms"`
	FlatRegions            string `json:"f_regions"`
	FlatMissions           string `json:"f_missions"`
	FlatShare              string `json:"f_share"`
}

// RawSnippetFields is a struct for an individual snippet field
type RawSnippetFields struct {
	FieldName string   `json:"field"`
	Treatment string   `json:"treatment"`
	Values    []string `json:"values"`
}

func (fields *RawSnippetFields) String() string {
	return fmt.Sprintf("%s %s (%s)", fields.FieldName, fields.Treatment, strings.Join(fields.Values, ","))
}

// ODriveRawSnippetFields is a struct holding an array of snippet fields
type ODriveRawSnippetFields struct {
	Snippets []RawSnippetFields
}

func (snippets *ODriveRawSnippetFields) String() string {
	o := []string{}
	for _, s := range snippets.Snippets {
		o = append(o, s.String())
	}
	sort.Strings(o)
	return strings.Join(o, " AND ")
}

// NewODriveRawSnippetFieldFromString takes an individual fields snippet in escaped format and returns the structured object for the snippet
func newODriveRawSnippetFieldFromString(quotedSnippet string, expectedFieldName string) (RawSnippetFields, error) {

	// oDrive no longer trusts AAC to provide back our snippet in the expected
	// format. As such, this portion of the snippet parsed needs to be validated
	// to ensure that the (1) quotedSnippet is not 'null', or an empty string, and
	// that the resulting (2) field name matches the expected value for this snippet
	// and that (3) the value for treatment is an acceptable value.

	var rawSnippetFields RawSnippetFields
	var err error

	// Validate quotedSnippet is not null
	if strings.Compare(quotedSnippet, "null") == 0 {
		logger.Error("acm snippet field is null", zap.Object("acm", quotedSnippet))
		return rawSnippetFields, fmt.Errorf("AAC returned a snippet where %s is null", expectedFieldName)
	}
	// Validate quotedSnippet is not an empty string
	if len(strings.TrimSpace(quotedSnippet)) == 0 {
		logger.Error("acm snippet field is empty", zap.Object("acm", quotedSnippet))
		return rawSnippetFields, fmt.Errorf("AAC returned a snippet where %s is empty", expectedFieldName)
	}

	// Parse the snippet into the struct
	jsonIOReader := bytes.NewBufferString(quotedSnippet)
	err = (json.NewDecoder(jsonIOReader)).Decode(&rawSnippetFields)
	if err != io.EOF && err != nil {
		logger.Error("acm snippet field unparseable", zap.Object("acm", quotedSnippet), zap.String("err", err.Error()))
		return rawSnippetFields, err
	}
	if err == io.EOF {
		logger.Warn("acm snippet field empty")
	}

	// Validate the snippet field name matches the expected field name
	if strings.Compare(expectedFieldName, rawSnippetFields.FieldName) != 0 {
		logger.Error("acm snippet field name mismatch", zap.Object("expectedFieldName", expectedFieldName), zap.Object("rawSnippetFields.FieldName", rawSnippetFields.FieldName))
		return rawSnippetFields, fmt.Errorf("AAC returned a snippet where field name %s did not match expected name %s", rawSnippetFields.FieldName, expectedFieldName)
	}

	// Validate the snippet treatment type is a supported  value
	switch rawSnippetFields.Treatment {
	case "allowed", "disallow": // valid
	default:
		logger.Error("acm snippet field treatment type is unsupported", zap.Object("rawSnippetFields.Treatment", rawSnippetFields.Treatment))
		return rawSnippetFields, fmt.Errorf("AAC returned a snippet where treatment type %s is not supported on field %s", rawSnippetFields.Treatment, expectedFieldName)
	}

	return rawSnippetFields, nil
}

// NewODriveRawSnippetFieldsFromSnippetResponse takes the entire snippet response from AAC and builds the raw snippet fields array
func NewODriveRawSnippetFieldsFromSnippetResponse(snippets string) (ODriveRawSnippetFields, error) {
	var rawSnippetFields ODriveRawSnippetFields
	var err error

	//The snippets may be escape-quoted. If so, resolve this
	unquotedSnippets := snippets
	if strings.HasPrefix(snippets, `{\`) {
		unquotedSnippets, err = strconv.Unquote(snippets)
		if err != nil {
			logger.Error("acm snippet unquoting error", zap.Object("snippets", snippets), zap.Object("err", err.Error()))
			return rawSnippetFields, err
		}
	}

	// oDrive no longer trusts AAC to provide back our snippet in the expected
	// format. As such, the response returned must be validated against our
	// expectations to be considered valid.  If not valid, an error will be
	// returned and its up to server handlers to return 403 Unauthorized.
	// If at any point we fail on an individual field, fail fast. Don't continue
	// processing the others as it means AAC is not properly configured and
	// nothing about this snippet should be trusted.

	allowSnippetsToBeDynamic := true

	if allowSnippetsToBeDynamic {
		return convertSnippetsUsingInterface(unquotedSnippets)
	}
	return convertSnippetsUsingStruct(unquotedSnippets)
}

func convertSnippetsUsingInterface(snippets string) (ODriveRawSnippetFields, error) {
	var rawSnippetFields ODriveRawSnippetFields
	var snippet RawSnippetFields
	var err error

	// Stage 1: Convert snippets to a map
	snippetInterface, err := utils.UnmarshalStringToInterface(snippets)
	if err != nil {
		logger.Error("acm snippet unparseable", zap.Object("snippets", snippets), zap.Object("err", err.Error()))
		return rawSnippetFields, err
	}
	snippetMap, ok := snippetInterface.(map[string]interface{})
	if !ok {
		logger.Error("acm snippet does not convert to map", zap.Object("snippets", snippets))
		return rawSnippetFields, fmt.Errorf("Unable to process snippet for user")
	}

	// Stage 2: Iterate the map, verifying that each of the keys has a value
	// that parses against the RawSnippetFields struct and expected field names are
	// met
	for snippetKey, snippetValue := range snippetMap {
		// Determine the expected name. Due to AAC bugs, the key's f_sap and f_aea had
		// to be present in the definition, and then mapped back to target fields.
		// Note that the source (key) ultimately does not matter, while the
		// expectedFieldName ends up being stored in the DB elsewhere, and used for
		// comparisons and should equate to the flattened field name in an object ACM.
		// If AAC is ever fixed and odrive-raw.yml updated such that f_sar_id is
		// used in place of f_sap, then in theory this should still be able to work.
		expectedFieldName := snippetKey
		switch snippetKey {
		case "f_sap":
			expectedFieldName = "f_sar_id"
		case "f_aea":
			expectedFieldName = "f_atom_energy"
		default:
			expectedFieldName = snippetKey
		}

		// Serialize the value to a string
		snippetPart, err := utils.MarshalInterfaceToString(snippetValue)
		if err != nil {
			logger.Error("acm snippet part could not be serialized", zap.Object("key", snippetKey), zap.Object("snippetValue", snippetValue))
			return rawSnippetFields, fmt.Errorf("Unable to process snippet for user. Field %s could not be parsed", snippetKey)
		}

		// Parse to our opinionated struct with validation checks
		snippet, err = newODriveRawSnippetFieldFromString(snippetPart, expectedFieldName)
		if err != nil {
			return rawSnippetFields, err
		}

		// Add it to the processed snippets that will be returned
		rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	}

	return rawSnippetFields, nil
}

func convertSnippetsUsingStruct(snippets string) (ODriveRawSnippetFields, error) {
	var parsedSnippets ODriveRawSnippet
	var snippet RawSnippetFields
	var rawSnippetFields ODriveRawSnippetFields
	var err error

	// Stage 1: Verify that the first pass can be parsed out in to the expected
	// fields defined in the ODriveRawSnippet struct above.
	jsonIOReader := bytes.NewBufferString(snippets)
	err = (json.NewDecoder(jsonIOReader)).Decode(&parsedSnippets)
	if err != nil {
		logger.Error("acm snippet unparseable", zap.Object("snippets", snippets))
		return rawSnippetFields, err
	}

	// Stage 2: Verify that each of the expected fields returned is not null,
	// has a value that when parsed, meets expected naming and other conditions

	// f_macs
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatMACs, "f_macs")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_oc_org
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatOCOrgs, "f_oc_org")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_sap -> f_sar_id
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatSAP, "f_sar_id")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_clearance
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatClearance, "f_clearance")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_aea -> f_atom_energy
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatAtomEnergy, "f_atom_energy")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_sci_ctrls
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatSCIControls, "f_sci_ctrls")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// dissem_countries
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.DisseminationCountries, "dissem_countries")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_accms
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatACCMs, "f_accms")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_regions
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatRegions, "f_regions")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_missions
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatMissions, "f_missions")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	// f_share
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatShare, "f_share")
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)

	return rawSnippetFields, nil
}
