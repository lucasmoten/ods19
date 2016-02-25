package models

// ODCommonMeta is a nestable structure defining the attributes most common for
// Object Drive elements
type ODCommonMeta struct {
	ODID
	ODCreatable
	ODModifiable
	ODDeletable
}

// NewODCommonMetaWithDN ...
func NewODCommonMetaWithDN(dn string) (ODCommonMeta, error) {
	var common ODCommonMeta
	// NewODID
	id, err := NewODID()
	if err != nil {
		return common, err
	}
	common.ID = id.ID
	// NewCreateable
	cr := NewoODCreateableWithDN(dn)
	common.CreatedDate = cr.CreatedDate
	common.CreatedBy = cr.CreatedBy
	// NewODModifiable. Manually assign ModifiedBy and leave ModifiedDate blank.
	common.ModifiedBy = dn
	// NewODeletable
	common.IsDeleted = false
	return common, nil
}
