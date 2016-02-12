package dao

import "log"

// GetUsers retrieves all users.
func (dao *DataAccessLayer) GetUsers() ([]string, error) {
	// TODO this should return a User struct
	//XXX this is no good when the list is very large!!!!
	var result []string
	getUsersStatement := `select distinguishedName from user`
	err := dao.MetadataDB.Select(&result, getUsersStatement)
	if err != nil {
		log.Printf("Unable to execute query %s:%v", getUsersStatement, err)
	}
	return result, err
}
