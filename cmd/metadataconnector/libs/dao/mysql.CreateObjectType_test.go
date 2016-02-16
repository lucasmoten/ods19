package dao_test

// import (
// 	"database/sql"
// 	"testing"
//
// 	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
// 	"decipher.com/oduploader/metadata/models"
// )
//
// func _TestCreateObjectType(t *testing.T) {
// 	var objectType models.ODObjectType
// 	objectType.Name = "Test Type"
// 	objectType.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
//
// 	dbObjectType, err := dao.GetObjectTypeByName(db, objectType.Name, false, objectType.CreatedBy)
// 	objectTypeCreated := false
// 	// we can have an error if the object type not present
// 	if err != nil {
// 		// but it has to be a no rows error. anything else, fails the test
// 		if err != sql.ErrNoRows {
// 			t.Error(err)
// 		}
// 	}
// 	if dbObjectType.ID == nil {
// 		dao.CreateObjectType(db, &objectType)
// 		objectTypeCreated = true
// 	} else {
// 		objectType = dbObjectType
// 	}
//
// 	if objectType.ID == nil {
// 		t.Error("expected ID to be set")
// 	}
// 	if objectType.ModifiedBy != objectType.CreatedBy {
// 		t.Error("expected ModifiedBy to match CreatedBy")
// 	}
//
// 	if objectTypeCreated {
// 		err = dao.DeleteObjectType(db, &objectType)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 	}
// }
