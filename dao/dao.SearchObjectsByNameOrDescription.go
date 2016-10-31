package dao

import (
	"log"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
)

// SearchObjectsByNameOrDescription retrieves a list of Objects, their
// Permissions and optionally Properties in object drive that are
// available to the user making the call, matching any specified
// filter settings on the paging request, and ordered by sort settings
func (dao *DataAccessLayer) SearchObjectsByNameOrDescription(user models.ODUser, pagingRequest PagingRequest, loadProperties bool) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := searchObjectsByNameOrDescriptionInTransaction(tx, user, pagingRequest, loadProperties)
	if err != nil {
		dao.GetLogger().Error("Error in SearchObjectsByNameOrDescription", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func searchObjectsByNameOrDescriptionInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest, loadProperties bool) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}

	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
	query := `
    select 
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
        ,o.containsUSPersonsData
        ,o.exemptFromFOIA        
        ,ot.name typeName     
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op	on op.objectid = o.id
        inner join objectacm acm on o.id = acm.objectid
    where 
        o.isdeleted = 0 
        and op.isdeleted = 0
        and op.allowread = 1
        and o.isexpunged = 0 
        and o.isancestordeleted = 0`
	query += buildFilterForUserACMShare(user)
	query += buildFilterForUserSnippets(user)
	query += buildFilterSortAndLimit(pagingRequest)

	//log.Println(query)
	err := tx.Select(&response.Objects, query)
	if err != nil {
		return response, err
	}

	// Paging stats guidance
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
	response.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)

	// Each record in this page of results....
	for i := 0; i < len(response.Objects); i++ {
		// Permissions
		permissions, err := getPermissionsForObjectInTransaction(tx, response.Objects[i])
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Permissions = permissions
		// Properties
		if loadProperties {
			properties, err := getPropertiesForObjectInTransaction(tx, response.Objects[i])
			if err != nil {
				print(err.Error())
				return response, err
			}
			response.Objects[i].Properties = properties
		}
	}

	// Done
	return response, err
}

// buildOrderBy is a function intended for use in building up the base order by clause for
// all list type operations with sanity checks on the fieldnames past in so that we dont
// just take values straight from the caller/client.  To apply the order by to the
// archive table, call buildOrderByArchive
func buildOrderBy(pagingRequest PagingRequest) string {
	out := ` order by `
	if len(pagingRequest.SortSettings) > 0 {
		for _, sortSetting := range pagingRequest.SortSettings {

			dbField := getDBFieldFromPagingRequestField(sortSetting.SortField)
			if len(dbField) == 0 {
				// unrecognized/unhandled field
				continue
			}
			out += ` ` + dbField
			if sortSetting.SortAscending {
				out += ` asc,`
			} else {
				out += ` desc,`
			}
		}
		if strings.HasSuffix(out, ",") {
			out = strings.TrimRight(out, ",")
		}
	}
	if strings.Compare(out, " order by ") == 0 {
		// None of the sort settings matched. Force to default
		out += ` o.modifieddate desc`
	}
	return out
}

// buildOrderByArchive first builds up the order by clause from paging request,
// and then converts to reference the archive table
func buildOrderByArchive(pagingRequest PagingRequest) string {
	a := buildOrderBy(pagingRequest)
	a = asArchive(a)
	return a
}

func asArchive(in string) string {
	return strings.Replace(in, " o.", " ao.", -1)
}

func getDBFieldFromPagingRequestField(fieldName string) string {

	dbFields := map[string][]string{}
	dbFields["o.changecount"] = []string{"changecount", "version"}
	dbFields["o.createdby"] = []string{"createdby", "creator"}
	dbFields["o.createddate"] = []string{"createddate", "created_dt"}
	dbFields["o.containsuspersonsdata"] = []string{"containsuspersonsdata", "uspersons", "isusperson"}
	dbFields["o.contentsize"] = []string{"contentsize", "size"}
	dbFields["o.contenttype"] = []string{"contenttype", "mimetype"}
	dbFields["o.description"] = []string{"description", "abstract"}
	dbFields["o.exemptfromfoia"] = []string{"exemptfromfoia", "foiaexempt", "isfoiaexempt"}
	dbFields["o.id"] = []string{"id"}
	dbFields["o.modifiedby"] = []string{"modifiedby", "modifier"}
	dbFields["o.modifieddate"] = []string{"modifieddate", "date", "updated_dt"}
	dbFields["o.name"] = []string{"name", "title"}
	dbFields["o.ownedby"] = []string{"ownedby", "owner"}
	dbFields["ot.name"] = []string{"typename", "type", "kind"}

	field := strings.ToLower(strings.TrimSpace(fieldName))

	for dbField, aliases := range dbFields {
		for _, alias := range aliases {
			if field == alias {
				return dbField
			}
		}
	}

	return ""
}

func buildFilter(pagingRequest PagingRequest) string {
	out := ``
	if len(pagingRequest.FilterSettings) > 0 {
		matchType := " or "
		if pagingRequest.FilterMatchType == "and" {
			matchType = " and "
		}
		for _, filterSetting := range pagingRequest.FilterSettings {
			dbField := getDBFieldFromPagingRequestField(filterSetting.FilterField)
			if len(dbField) == 0 {
				// unrecognized/unhandled field
				continue
			}
			out += matchType + dbField
			switch strings.ToLower(filterSetting.Condition) {
			case "contains":
				out += ` like '%` + MySQLSafeString(filterSetting.Expression) + `%'`
			default: // "equals":
				out += ` like '` + MySQLSafeString(filterSetting.Expression) + `'`
			}
		}
		if len(out) > 0 {
			// Replace only the first condition with the group opener and close out the group
			out = strings.Replace(out, matchType, ` and (`, 1) + `)`
		}
	}
	return out
}

func buildFilterSortAndLimit(pagingRequest PagingRequest) string {
	limit := GetLimit(pagingRequest.PageNumber, pagingRequest.PageSize)
	offset := GetOffset(pagingRequest.PageNumber, pagingRequest.PageSize)
	sqlStatementSuffix := ``
	sqlStatementSuffix += buildFilter(pagingRequest)
	sqlStatementSuffix += buildOrderBy(pagingRequest)
	sqlStatementSuffix += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	return sqlStatementSuffix
}
func buildFilterSortAndLimitArchive(pagingRequest PagingRequest) string {
	a := buildFilterSortAndLimit(pagingRequest)
	a = strings.Replace(a, " o.", " ao.", -1)
	return a
}

func buildFilterForUserACMShare(user models.ODUser) string {

	// Return if user.Snippets not defined, or there were no shares.
	defaultSQL := " and op.grantee = '" + MySQLSafeString(user.DistinguishedName) + "' "

	if user.Snippets == nil {
		return defaultSQL
	}

	// sql is going to be the returned portion of the where clause built up from the snippets
	var sql string

	// If the snippet defines f_share, this will be used to capture as we iterate through the fields
	var shareSnippet acm.RawSnippetFields

	// Now iterate all the fields looking for f_share
	for _, rawFields := range user.Snippets.Snippets {
		switch rawFields.FieldName {
		case "f_share":
			// This field is handled differently tied into permissions.  Capture it
			shareSnippet = rawFields
			break
		default:
			// All other snippet fields ignored for this operation
			continue
		}
	}

	// If share settings were defined with additional groups
	if len(shareSnippet.Values) > 0 {
		sql = " and (op.grantee = '" + MySQLSafeString(user.DistinguishedName) + "' or op.grantee like '" + MySQLSafeString(models.AACFlatten(models.EveryoneGroup)) + "'"
		for _, shareValue := range shareSnippet.Values {
			//if !strings.Contains(shareValue, "cusou") && !strings.Contains(shareValue, "governmentcus") {
			sql += " or op.grantee like '" + MySQLSafeString(shareValue) + "'"
			//}
		}
		sql += ") "
	} else {
		sql = defaultSQL
	}
	//log.Printf(sql)
	return sql
}

func buildFilterForUserSnippets(user models.ODUser) string {

	if user.Snippets == nil {
		return " "
	}

	return buildFilterForUserSnippetsUsingACM(user)
}

func buildFilterForUserSnippetsUsingACM(user models.ODUser) string {
	if user.Snippets == nil {
		return " "
	}

	// sql is going to be the returned portion of the where clause built up from the snippets
	var sql string

	// table alias 'acm' refers to 'objectacm', a join between object and acm consisting of parts

	sql += " and acm.acmId in (select acmid from acm where 1=1 "

	// Now iterate all the fields building up the where clause portion
	for _, rawFields := range user.Snippets.Snippets {
		switch rawFields.Treatment {
		case "disallow":
			sql += " and acmid not in ("
			// where it does have the field
			sql += "select acmid from acmpart inner join acmkey on acmpart.acmkeyid = acmkey.id inner join acmvalue on acmpart.acmvalueid = acmvalue.id "
			sql += "where acmkey.name like '" + MySQLSafeString(rawFields.FieldName) + "' "
			sql += "and acmvalue.name in (''"
			for _, value := range rawFields.Values {
				sql += ",'" + MySQLSafeString2(value) + "'"
			}
			sql += ") and acmpart.isdeleted = 0 and acmkey.isdeleted = 0 and acmvalue.isdeleted = 0) "
		case "allowed":
			sql += " and acmid in ("
			// where it doesn't have the field
			sql += "select id from acm where isdeleted = 0 and id not in (select acm.id from acm inner join acmpart on acmpart.acmid = acm.id and acmpart.isdeleted = 0 inner join acmkey on acmpart.acmkeyid = acmkey.id and acmkey.name like '" + MySQLSafeString(rawFields.FieldName) + "' and acmkey.isdeleted = 0 where acm.isdeleted = 0)"
			// where it does have the field
			sql += " union "
			sql += "select acmid from acmpart inner join acmkey on acmpart.acmkeyid = acmkey.id inner join acmvalue on acmpart.acmvalueid = acmvalue.id "
			sql += "where acmkey.name like '" + MySQLSafeString(rawFields.FieldName) + "' "
			sql += "and (acmvalue.name = '' "
			for _, value := range rawFields.Values {
				sql += " or acmvalue.name like '" + MySQLSafeString(value) + "'"
			}
			sql += ") and acmpart.isdeleted = 0 and acmkey.isdeleted = 0 and acmvalue.isdeleted = 0)"
		default:
			log.Printf("Warning: Unhandled treatment type from snippets")
		}
	}

	sql += ")"
	//log.Printf(sql)
	return sql
}

func buildFilterExcludeEveryone() string {
	var sql string
	sql += " and o.id not in ("
	sql += "select objectid from object_permission "
	sql += "where isdeleted = 0 and grantee like '"
	sql += MySQLSafeString(models.AACFlatten(models.EveryoneGroup))
	sql += "') "
	return sql
}

// MySQLSafeString takes an input string and escapes characters as appropriate
// to make it safe for usage as a string input when building dynamic sql query
// Based upon: https://www.owasp.org/index.php/SQL_Injection_Prevention_Cheat_Sheet#MySQL_Escaping
func MySQLSafeString(i string) string {
	o := ""
	b := []byte(i)
	for _, v := range b {
		switch v {
		case 0x00: // NULL
			o += `\0`
		case 0x08: // Backspace
			o += `\b`
		case 0x09: // Tab
			o += `\t`
		case 0x0a: // Linefeed
			o += `\n`
		case 0x0d: // Carriage Return
			o += `\r`
		case 0x1a: // Substitute Character
			o += `\Z`
		case 0x22: // Double Quote
			o += `\"`
		case 0x25: // Percent Symbol
			o += `\%`
		case 0x27: // Single Quote
			o += `\'`
		case 0x5c: // Backslash
			o += `\\`
		case 0x5f: // Underscore
			o += `\_`
		default:
			o += string(v)
		}

	}
	return o
}

// MySQLSafeString2 takes an input string and escapes characters as appropriate
// to make it safe for usage as a string input when building dynamic sql query
// Based upon: https://www.owasp.org/index.php/SQL_Injection_Prevention_Cheat_Sheet#MySQL_Escaping
// With the following EXCEPTION!!!!! -- This does not escape underscores
func MySQLSafeString2(i string) string {
	o := ""
	b := []byte(i)
	for _, v := range b {
		switch v {
		case 0x00: // NULL
			o += `\0`
		case 0x08: // Backspace
			o += `\b`
		case 0x09: // Tab
			o += `\t`
		case 0x0a: // Linefeed
			o += `\n`
		case 0x0d: // Carriage Return
			o += `\r`
		case 0x1a: // Substitute Character
			o += `\Z`
		case 0x22: // Double Quote
			o += `\"`
		case 0x25: // Percent Symbol
			o += `\%`
		case 0x27: // Single Quote
			o += `\'`
		case 0x5c: // Backslash
			o += `\\`
		// case 0x5f: // Underscore
		// 	o += `\_`
		default:
			o += string(v)
		}

	}
	return o
}

func buildFilterRequireObjectsIOwn(user models.ODUser) string {
	return " and o.ownedby in (" + strings.Join(buildListObjectsIOwn(user), ",") + ")"
}

func buildFilterExcludeObjectsIOrMyGroupsOwn(tx *sqlx.Tx, user models.ODUser) string {
	return " and o.ownedby not in (" + strings.Join(buildListObjectsIOrMyGroupsOwn(tx, user), ",") + ")"
}

func buildFilterRequireObjectsIOrMyGroupsOwn(tx *sqlx.Tx, user models.ODUser) string {
	return " and o.ownedby in (" + strings.Join(buildListObjectsIOrMyGroupsOwn(tx, user), ",") + ")"
}

func buildListObjectsIOrMyGroupsOwn(tx *sqlx.Tx, user models.ODUser) []string {
	ownedby := []string{}
	ownedby = append(ownedby, buildListObjectsIOwn(user)...)
	ownedby = append(ownedby, buildListObjectsMyGroupOwns(tx, user)...)
	return ownedby
}

func buildListObjectsIOwn(user models.ODUser) []string {
	ownedby := []string{}
	// original ownership format as just the dn
	ownedby = append(ownedby, "'"+MySQLSafeString2(user.DistinguishedName)+"'")
	// revised resource format prefixed by type
	ownedby = append(ownedby, "'user/"+MySQLSafeString2(user.DistinguishedName)+"'")
	return ownedby
}
func buildListObjectsMyGroupOwns(tx *sqlx.Tx, user models.ODUser) []string {
	ownedby := []string{}
	if user.Snippets != nil {
		for _, rawFields := range user.Snippets.Snippets {
			switch rawFields.FieldName {
			case "f_share":
				for _, shareValue := range rawFields.Values {
					trimmedShareValue := strings.TrimSpace(shareValue)
					if len(trimmedShareValue) > 0 {
						acmGrantee, err := getAcmGranteeInTransaction(tx, trimmedShareValue)
						if err == nil {
							ownedby = append(ownedby, "'"+acmGrantee.ResourceName()+"'")
						} else {
							// Assume its a group
							ownedby = append(ownedby, "'group/"+MySQLSafeString2(trimmedShareValue)+"'")
						}
					}
				}
				break
			default:
				continue
			}
		}
	}
	return ownedby
}
