package testhelpers

import "decipher.com/oduploader/metadata/models"

// NewODCommonMetaWithDN is a convenince constructor for creating an ODCommonMeta.
func NewODCommonMetaWithDN(dn string) (models.ODCommonMeta, error) {
	var common models.ODCommonMeta
	id, err := models.NewODID()
	if err != nil {
		return common, err
	}
	common.ID = id.ID
	cr := models.NewoODCreateableWithDN(dn)
	common.CreatedDate = cr.CreatedDate
	common.CreatedBy = cr.CreatedBy
	common.IsDeleted = false
	return common, nil
}
