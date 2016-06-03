package acm

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/uber-go/zap"

	globalconfig "decipher.com/object-drive-server/config"
)

var (
	logger = globalconfig.RootLogger
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

// ODriveRawSnippetFields is a struct holding an array of snippet fields
type ODriveRawSnippetFields struct {
	Snippets []RawSnippetFields
}

// NewODriveRawSnippetFieldFromString takes an individual fields snippet in escaped format and returns the structured object for the snippet
func newODriveRawSnippetFieldFromString(quotedSnippet string) (RawSnippetFields, error) {
	var rawSnippetFields RawSnippetFields
	var err error
	jsonIOReader := bytes.NewBufferString(quotedSnippet)

	// log.Printf(quotedSnippet)
	// unquotedSnippet, err = strconv.Unquote(quotedSnippet)
	// if err != nil {
	// 	log.Printf(err.Error())
	// }
	// log.Printf(unquotedSnippet)
	// if err != nil {
	// 	return rawSnippetFields, nil
	// }
	// jsonIOReader := bytes.NewBufferString(unquotedSnippet)

	err = (json.NewDecoder(jsonIOReader)).Decode(&rawSnippetFields)
	if err != io.EOF && err != nil {
		logger.Error("acm snippet unparseable", zap.Object("acm", quotedSnippet), zap.String("err", err.Error()))
		return rawSnippetFields, err
	}
	if err == io.EOF {
		logger.Warn("acm snippet empty")
	}
	return rawSnippetFields, nil
}

// NewODriveRawSnippetFieldsFromSnippetResponse takes the entire snippet response from AAC and builds the raw snippet fields array
func NewODriveRawSnippetFieldsFromSnippetResponse(snippets string) (ODriveRawSnippetFields, error) {
	var parsedSnippets ODriveRawSnippet
	var snippet RawSnippetFields
	var rawSnippetFields ODriveRawSnippetFields
	jsonIOReader := bytes.NewBufferString(snippets)
	err := (json.NewDecoder(jsonIOReader)).Decode(&parsedSnippets)
	if err != nil {
		return rawSnippetFields, err
	}

	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatMACs)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatOCOrgs)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatSAP)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatClearance)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatAtomEnergy)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatSCIControls)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.DisseminationCountries)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatACCMs)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatRegions)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatMissions)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)
	snippet, err = newODriveRawSnippetFieldFromString(parsedSnippets.FlatShare)
	if err != nil {
		return rawSnippetFields, err
	}
	rawSnippetFields.Snippets = append(rawSnippetFields.Snippets, snippet)

	return rawSnippetFields, nil
}
