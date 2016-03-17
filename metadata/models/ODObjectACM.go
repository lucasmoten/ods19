package models

/*
ODObjectACM is a structure defining the flattened attributes for ACM in the database
*/
type ODObjectACM struct {
	ODCommonMeta
	// ObjectID references the object identifier for which this ACM applies
	ObjectID []byte `db:"objectId"`
	// ACMID references the ACM identifier for which this ACM applies
	ACMID               []byte     `db:"acmId"`
	FlatClearance       string     `db:"f_clearance"`
	FlatShare           NullString `db:"f_share"`
	FlatOCOrgs          NullString `db:"f_oc_org"`
	FlatMissions        NullString `db:"f_missions"`
	FlatRegions         NullString `db:"f_regions"`
	FlatMAC             NullString `db:"f_macs"`
	FlatSCI             NullString `db:"f_sci_ctrls"`
	FlatACCMS           NullString `db:"f_accms"`
	FlatSAR             NullString `db:"f_sar_id"`
	FlatAtomEnergy      NullString `db:"f_atom_energy"`
	FlatDissemCountries NullString `db:"f_dissem_countries"`
}
