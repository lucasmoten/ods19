package models

// ODCommonMeta is a nestable structure defining the attributes most common for
// Object Drive elements
type ODCommonMeta struct {
	ODID
	ODCreatable
	ODModifiable
	ODDeletable
}

// NewODCommonMetaWithDN is a convenince constructor for creating an ODCommonMeta.
func NewODCommonMetaWithDN(dn string) (ODCommonMeta, error) {
	var common ODCommonMeta
	id, err := NewODID()
	if err != nil {
		return common, err
	}
	common.ID = id.ID
	cr := NewoODCreateableWithDN(dn)
	common.CreatedDate = cr.CreatedDate
	common.CreatedBy = cr.CreatedBy
	common.IsDeleted = false
	return common, nil
}
