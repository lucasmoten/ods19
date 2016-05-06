package dao

import (
	"database/sql"
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObject uses the passed in object and makes the appropriate sql calls to
// the database to retrieve and return the requested object by ID. Optionally,
// loadProperties flag pulls in nested properties associated with the object.
func (dao *DataAccessLayer) GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error) {

	tx := dao.MetadataDB.MustBegin()
	dbObject, err := getObjectInTransaction(tx, object, loadProperties)
	if err != nil {
		log.Printf("Error in GetObject: %v\n", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObject, err
}

func getObjectInTransaction(tx *sqlx.Tx, object models.ODObject, loadProperties bool) (models.ODObject, error) {
	var dbObject models.ODObject

	if len(object.ID) == 0 {
		return dbObject, ErrMissingID
	}

	getObjectStatement := `
    select 
        o.id    
        ,o.createdDate
        ,o.createdBy
        ,o.modifiedDate
        ,o.modifiedBy
        ,o.isDeleted
        ,o.deletedDate
        ,o.deletedBy
        ,o.isAncestorDeleted
        ,o.isExpunged
        ,o.expungedDate
        ,o.expungedBy
        ,o.changeCount
        ,o.changeToken
        ,o.ownedBy
        ,o.typeId
        ,o.name
        ,o.description
        ,o.parentId
        ,o.contentConnector
        ,o.rawAcm
        ,o.contentType
        ,o.contentSize
        ,o.contentHash
        ,o.encryptIV
        ,o.ownedByNew
        ,o.isPDFAvailable
        ,o.isStreamStored
        ,o.isUSPersonsData
        ,o.isFOIAExempt
        ,ot.name typeName     
    from object o 
        inner join object_type ot on o.typeid = ot.id 
    where o.id = ?`
	err := tx.Unsafe().Get(&dbObject, getObjectStatement, object.ID)
	if err != nil {
		return dbObject, err
	}

	// Load Permissions
	dbPermissions, dbPermErr := getPermissionsForObjectInTransaction(tx, object)
	dbObject.Permissions = dbPermissions
	if dbPermErr != nil {
		err = dbPermErr
		return dbObject, err
	}
	// Load ACM structure
	dbACM, dbACMErr := getObjectACMForObjectInTransaction(tx, object, true)
	dbObject.ACM = dbACM
	if dbACMErr != nil {
		err = dbACMErr
		return dbObject, err
	}
	// Load properties if requested
	if loadProperties {
		dbProperties, dbPropErr := getPropertiesForObjectInTransaction(tx, object)
		dbObject.Properties = dbProperties
		if dbPropErr != nil {
			err = dbPropErr
			return dbObject, err
		}
	}

	// Done
	return dbObject, nil
}

func getObjectACMForObjectInTransaction(tx *sqlx.Tx, object models.ODObject, createIfMissing bool) (models.ODObjectACM, error) {
	var dbObjectACM models.ODObjectACM

	getStatement := `
    select 
        oa.id    
        ,oa.createdDate
        ,oa.createdBy
        ,oa.modifiedDate
        ,oa.modifiedBy
        ,oa.isDeleted
        ,oa.deletedDate
        ,oa.deletedBy
        ,oa.objectId
        ,oa.acmId
        ,oa.f_clearance
        ,oa.f_share
        ,oa.f_oc_org
        ,oa.f_missions
        ,oa.f_regions
        ,oa.f_macs
        ,oa.f_sci_ctrls
        ,oa.f_accms
        ,oa.f_sar_id
        ,oa.f_atom_energy
        ,oa.f_dissem_countries 
    from object_acm oa 
    where 
        oa.isdeleted = 0 
        and oa.objectId = ? 
    order by oa.createddate desc limit 1`
	err := tx.Unsafe().Get(&dbObjectACM, getStatement, object.ID)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			// No ACM saved in this object yet.
			if !createIfMissing {
				return dbObjectACM, err
			}
			dbACM, dbACMErr := createObjectACMForObjectInTransaction(tx, &object)
			dbObjectACM = dbACM
			if dbACMErr != nil {
				err = dbACMErr
				return dbObjectACM, err
			}
		default:
			return dbObjectACM, err
		}
	}

	// Done
	return dbObjectACM, nil
}
