package testhelpers

import "decipher.com/object-drive-server/services/aac"

func GetTestSnippetResponse() *aac.SnippetResponse {

	snippets := `{"success":true,"messages":null,"snippets":"{\"f_macs\":\"{\\\"field\\\":\\\"f_macs\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"tide\\\",\\\"bir\\\",\\\"watchdog\\\"]}\",\"f_oc_org\":\"{\\\"field\\\":\\\"f_oc_org\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dia\\\"]}\",\"f_accms\":\"{\\\"field\\\":\\\"f_accms\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[]}\",\"f_sap\":\"{\\\"field\\\":\\\"f_sar_id\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_clearance\":\"{\\\"field\\\":\\\"f_clearance\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"ts\\\",\\\"s\\\",\\\"c\\\",\\\"u\\\"]}\",\"f_regions\":\"{\\\"field\\\":\\\"f_regions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_missions\":\"{\\\"field\\\":\\\"f_missions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_share\":\"{\\\"field\\\":\\\"f_share\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"cntesttester10oupeopleoudaeouchimeraou_s_governmentcus\\\",\\\"cusou_s_governmentouchimeraoudaeoupeoplecntesttester10\\\"]}\",\"f_aea\":\"{\\\"field\\\":\\\"f_atom_energy\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_sci_ctrls\":\"{\\\"field\\\":\\\"f_sci_ctrls\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"kdk\\\",\\\"rsv\\\"]}\",\"dissem_countries\":\"{\\\"field\\\":\\\"dissem_countries\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"USA\\\"]}\"}"}`

	// Messages is unset on SnippetResponse for now.
	snippetResp := &aac.SnippetResponse{Success: true, Snippets: snippets}

	return snippetResp
}
