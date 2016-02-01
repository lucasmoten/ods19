/* ACM thrift object definition */

namespace * gov.ic.dodiis.dctc.bedrock.audit.thrift

struct OCAttribs {
	1: optional list<string> missions,
	2: optional list<string> regions,
	3: optional list<string> orgs
}

struct Project {
	1: optional string disp_nm,
	2: optional list<string> groups
}

struct Share {
	1: optional list<string> users,
	2: optional map<string, Project> projects
}

struct CoiControl {
	1: optional string coi_ctrl,
	2: optional string disp_nm
}

struct Accm {
	1: optional string coi,
	2: optional string disp_nm,
	3: optional list<CoiControl> coi_ctrls
}

struct AssignedControls {
	1: optional list<string> coi,
	2: optional list<string> coi_ctrls
}

struct Acm {
	1: optional string version,
	2: required string classif,
	3: required list<string> owner_prod,
	4: optional list<string> atom_energy,
	5: optional list<string> non_us_ctrls,
	6: optional list<string> non_ic,
	7: optional list<string> sci_ctrls,
	8: optional list<string> sar_id,
	9: optional list<string> disponly_to,
	10: optional string disp_only,
	11: optional list<string> dissem_ctrls,
	12: optional list<string> rel_to,
	13: optional string classif_rsn,
	14: optional string classif_by,
	15: optional string deriv_from,
	16: optional string complies_with,
	17: optional string classif_dt,
	18: optional string declass_dt,
	19: optional string declass_event,
	20: optional string declass_ex,
	21: optional string deriv_class_by,
	22: optional string des_version,
	23: optional string notice_rsn,
	24: optional string poc,
	25: optional string rsrc_elem,
	26: optional string compil_rsn,
	27: optional string ex_from_rollup,
	28: optional list<string> fgi_open,
	29: optional list<string> fgi_protect,
	30: optional string portion,
	31: optional string banner,
	32: optional list<string> dissem_countries,
	33: optional list<OCAttribs> oc_attribs,
	34: optional list<Accm> accms,
	35: optional list<Accm> macs,
	36: optional AssignedControls assigned_controls,
	37: optional Share share,
	38: optional list<string> f_clearance,
	39: optional list<string> f_classif_rank,
	40: optional list<string> f_sci_ctrls,
	41: optional list<string> f_accms,
	42: optional list<string> f_oc_org,
	43: optional list<string> f_regions,
	44: optional list<string> f_missions,
	45: optional list<string> f_share,
	46: optional list<string> f_sar_id,
	47: optional list<string> f_atom_energy,
	48: optional list<string> f_macs
}