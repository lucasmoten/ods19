package acm

import (
	"bytes"
	"encoding/json"
)

// ACM is a structure modeled after the rawACM on an object.
type ACM struct {
	Version                 string            `json:"version"`
	Classif                 string            `json:"classif"`
	OwnerProducer           []string          `json:"owner_prod"`
	AtomEnergy              []string          `json:"atom_energy"`
	SpecialAccessRequiredID []string          `json:"sar_id"`
	SCIControls             []string          `json:"sci_ctrls"`
	DisplayOnlyTo           []string          `json:"disponly_to"`
	DisseminationControls   []string          `json:"dissem_ctrls"`
	NonICMarkings           []string          `json:"non_ic"`
	ReleasableTo            []string          `json:"rel_to"`
	FGIOpen                 []string          `json:"fgi_open"`
	FGIProtect              []string          `json:"fgi_protect"`
	PortionMark             string            `json:"portion"`
	OverallBanner           string            `json:"banner"`
	DisseminationCountries  []string          `json:"dissem_countries"`
	ACCMs                   []string          `json:"accms"`
	MACs                    []string          `json:"macs"`
	OCAttributes            []OCAttributeInfo `json:"oc_attribs"`
	FlatClearance           []string          `json:"f_clearance"`
	FlatSCIControls         []string          `json:"f_sci_ctrls"`
	FlatACCMs               []string          `json:"f_accms"`
	FlatOCOrgs              []string          `json:"f_oc_org"`
	FlatRegions             []string          `json:"f_regions"`
	FlatMissions            []string          `json:"f_missions"`
	FlatShare               []string          `json:"f_share"`
	FlatAtomEnergy          []string          `json:"f_atom_energy"`
	FlatMACs                []string          `json:"f_macs"`
	DisplayOnly             string            `json:"disp_only"`
}

// NewACMFromRawACM is a helper that takes a rawACM as may be presented with
// a request and unmarshals to an ACM object to isolate the different fields
func NewACMFromRawACM(rawACM string) (ACM, error) {
	var parsedACM ACM

	jsonACM := rawACM
	jsonIOReader := bytes.NewBufferString(jsonACM)
	err := (json.NewDecoder(jsonIOReader)).Decode(&parsedACM)
	if err != nil {
		return parsedACM, err
	}

	return parsedACM, nil
}
