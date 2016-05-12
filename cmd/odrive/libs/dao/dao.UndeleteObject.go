package dao

import (
	"database/sql"
	"errors"
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
)

// UndeleteObject undeletes an object at the database level, and propagates the
// undelete action to children that were not explicitly deleted themselves.
func (dao *DataAccessLayer) UndeleteObject(object *models.ODObject) (models.ODObject, error) {

	tx := dao.MetadataDB.MustBegin()
	dbObject, err := undeleteObjectInTransaction(tx, object)
	if err != nil {
		log.Printf("Error in UndeleteObject: %v\n", err)
		tx.Rollback()
		return models.ODObject{}, err
	}
	tx.Commit()
	return dbObject, nil
}

func undeleteObjectInTransaction(tx *sqlx.Tx, object *models.ODObject) (models.ODObject, error) {
	var dbObject models.ODObject

	if object.ID == nil {
		return dbObject, errors.New("Object ID was not specified for object being deleted")
	}
	if object.ChangeToken == "" {
		return dbObject, errors.New("Object ChangeToken was not specified for object being deleted")
	}

	undeleteStatement, err := tx.Prepare(`
    update object set modifiedBy = ?, 
        isdeleted = 0 where id = ?
    `)
	if err != nil {
		return dbObject, err
	}

	if _, err = undeleteStatement.Exec(object.ModifiedBy, object.ID); err != nil {
		return dbObject, err
	}

	err = undeleteAncestorChildren(tx, object)
	if err != nil {
		return dbObject, err
	}

	dbObject, err = getObjectInTransaction(tx, *object, false)

	return dbObject, nil
}

func undeleteAncestorChildren(tx *sqlx.Tx, object *models.ODObject) error {
	var results models.ODObjectResultset

	query := `select
        distinct sql_calc_found_rows 
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
    where 
        o.isancestordeleted = 1 
        and o.isexpunged = 0 
        and o.parentid = ?`

	err := tx.Select(&results.Objects, query, object.ID)
	if err != nil {
		log.Println("Error from Select in undeleteAncestorChildren: ", err)
		return err
	}

	// First, undelete the children.
	for _, child := range results.Objects {
		_, err := tx.Exec(`
		update object o 
		inner join object_permission op
		   on o.id = op.objectid
		set o.isancestordeleted = 0, o.isdeleted = 0
		where o.isexpunged = 0
		and o.id = ? `, child.ID)
		if err != nil {
			return err
		}
	}

	// Then, resursively call this function.
	for _, child := range results.Objects {
		err := undeleteAncestorChildren(tx, &child)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	}

	return nil

}
