package models

/*
ODACM is a structure defining the attributes for ACM.
*/
type ODACM struct {
	ODCommonMeta
	Classification            NullString `db:"classif"`
	OwnerProducer             string     `db:"owner_prod"`
	AtomicEnergy              NullString `db:"atom_energy"`
	NonUSControls             NullString `db:"non_us_ctrls"`
	SpecialAccessRequiredID   NullString `db:"sar_id"`
	SCIControls               NullString `db:"sci_ctrls"`
	DisseminationControls     NullString `db:"dissem_ctrls"`
	NonICMarkings             NullString `db:"non_ic"`
	ReleasbleTo               NullString `db:"rel_to"`
	ClassificationReason      NullString `db:"classif_rsn"`
	ClassifiedBy              NullString `db:"classif_by"`
	CompliesWith              NullString `db:"complies_with"`
	ClassificationDate        NullTime   `db:"classif_dt"`
	DeclassificationDate      NullTime   `db:"declass_dt"`
	DeclassificationEvent     NullString `db:"declass_event"`
	DeclassificationExemption NullString `db:"declass_ex"`
	DerivedClassificationBy   NullString `db:"deriv_class_by"`
	DerivedFrom               NullString `db:"deriv_from"`
	DESVersion                NullString `db:"des_version"`
	NoticeReason              NullString `db:"notice_rsn"`
	POC                       NullString `db:"poc"`
	ResourceElement           NullString `db:"rsrc_elem"`
	CompilReason              NullString `db:"compil_rsn"`
	ExemptedFromRollup        NullString `db:"ex_from_rollup"`
	FGIOpen                   NullString `db:"fgi_open"`
	FGIProtected              NullString `db:"fgi_protect"`
	PortionMark               NullString `db:"portion"`
	Banner                    NullString `db:"banner"`
	Version                   NullString `db:"version"`
	DisplayOnlyTo             NullString `db:"disponly_to"`
	DisseminationCountries    NullString `db:"dissem_countries"`
}
