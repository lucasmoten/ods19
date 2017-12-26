package dao

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
)

// SearchObjectsByNameOrDescription retrieves a list of Objects, their
// Permissions and optionally Properties in object drive that are
// available to the user making the call, matching any specified
// filter settings on the paging request, and ordered by sort settings
func (dao *DataAccessLayer) SearchObjectsByNameOrDescription(user models.ODUser, pagingRequest PagingRequest, loadProperties bool) (models.ODObjectResultset, error) {
	defer util.Time("SearchObjectsByNameOrDescription")()
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
        o.id    
    from object o
        inner join object_type ot on o.typeid = ot.id `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 0 and o.isexpunged = 0 and o.isancestordeleted = 0`
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query)
	if err != nil {
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, queryRowCount(query))
	if err != nil {
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
	response.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	// Load full meta, properties, and permissions
	for i := 0; i < len(response.Objects); i++ {
		obj, err := getObjectInTransaction(tx, response.Objects[i], loadProperties)
		if err != nil {
			return response, err
		}
		response.Objects[i] = obj
	}

	// Done
	return response, err
}

func queryRowCount(query string) string {
	lquery := strings.ToLower(query)
	// if precalculated - mysql/mariadb variants
	if strings.Index(lquery, "sql_calc_found_rows") > 0 {
		return "select found_rows()"
	}
	// inefficient ways of determining totals...
	// as simple select count
	queryCount := query
	if limitIdx := strings.Index(lquery, "limit "); limitIdx > 0 {
		queryCount = queryCount[:limitIdx]
	}
	if strings.Index(lquery, " count(") > 0 {
		// existing count. wrap the entire query.
		queryCount = "select count(0) from (" + lquery + ")"
	} else {
		// no nested count to account for
		if fromIdx := strings.Index(lquery, "from "); fromIdx > 0 {
			queryCount = "select count(0) " + queryCount[fromIdx:]
		}
	}
	return queryCount
}

// buildOrderBy is a function intended for use in building up the base order by clause for
// all list type operations with sanity checks on the fieldnames past in so that we dont
// just take values straight from the caller/client.  To apply the order by to the
// archive table, call buildOrderByArchive
func buildOrderBy(pagingRequest PagingRequest) string {
	sql := ` order by`
	useDefaultSort := true
	if len(pagingRequest.SortSettings) > 0 {
		for _, sortSetting := range pagingRequest.SortSettings {
			dbField := getDBFieldFromPagingRequestField(sortSetting.SortField)
			if len(dbField) == 0 {
				// skip this unrecognized/unhandled field
				continue
			}
			if !useDefaultSort {
				sql += ","
			}
			useDefaultSort = false
			sql += ` ` + dbField
			if sortSetting.SortAscending {
				sql += ` asc`
			} else {
				sql += ` desc`
			}
		}
	}
	if useDefaultSort {
		sql += ` o.modifieddate desc`
	}
	return sql
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
	dbFields["o.parentid"] = []string{"parentid", "parent", "folderid"}

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

			out += matchType
			isid := strings.HasSuffix(dbField, "id")
			if isid {
				out += ` hex(` + dbField + `) `
			} else {
				out += dbField
			}
			switch strings.ToLower(filterSetting.Condition) {
			case "morethan":
				out += ` > '` + MySQLSafeString(filterSetting.Expression) + `'`
			case "lessthan":
				out += ` < '` + MySQLSafeString(filterSetting.Expression) + `'`
			case "notbegins":
				out += ` not like '` + MySQLSafeString(filterSetting.Expression) + `%'`
			case "begins":
				out += ` like '` + MySQLSafeString(filterSetting.Expression) + `%'`
			case "notends":
				out += ` not like '%` + MySQLSafeString(filterSetting.Expression) + `'`
			case "ends":
				out += ` like '%` + MySQLSafeString(filterSetting.Expression) + `'`
			case "notcontains":
				out += ` not like '%` + MySQLSafeString(filterSetting.Expression) + `%'`
			case "contains":
				out += ` like '%` + MySQLSafeString(filterSetting.Expression) + `%'`
			case "notequals":
				out += ` not like '` + MySQLSafeString(filterSetting.Expression) + `'`
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

func buildJoinUserToACM(tx *sqlx.Tx, user models.ODUser) string {
	query := ` inner join acm2 on o.acmid = acm2.id inner join useracm on acm2.id = useracm.acmid and useracm.userid = unhex('`
	query += hex.EncodeToString(user.ID)
	query += `') `
	return query
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

func buildFilterRequireObjectsGroupOwns(tx *sqlx.Tx, groupGranteeName string) string {
	acmGrantee, err := getAcmGranteeInTransaction(tx, groupGranteeName)
	where := " and "
	if err != nil {
		// If there are no grantees matching this group, then no objects owned by it. Exclude everything
		where += "1 = 0"
	} else {
		where += "o.ownedby = '" + MySQLSafeString2(acmGrantee.ResourceNameRaw()) + "'"
	}
	return where
}

func removeDisplayNameFromResourceString(resourceString string) string {
	allButLastPart := resourceString[0:strings.LastIndex(resourceString, "/")]
	parts := strings.Split(resourceString, "/")
	partCount := len(parts)
	resourceType := parts[0]
	switch resourceType {
	case "user":
		switch partCount {
		// user/cn=bob
		case 2:
			return resourceString
		// user/cn=bob/bob
		case 3:
			return allButLastPart
		// invalid GIGO
		default:
			return resourceString
		}
	case "group":
		switch partCount {
		// group/_everyone
		case 2:
			return resourceString
		// group/_everyone/-Everyone
		// group/dctc/odrive
		case 3:
			// group/_everyone/-Everyone
			if models.AACFlatten(parts[1]) == models.AACFlatten(parts[2]) {
				return allButLastPart
			}
			return resourceString
		// group/dctc/DCTC/ODrive
		case 4:
			// TODO: When removing project display name, this will need altered
			// along with the calcResourceString database function
			return resourceString
		// group/dctc/DCTC/ODrive/DCTC ODrive
		case 5:
			return allButLastPart
		}
	// invalid GIGO
	default:
		return resourceString
	}
	// unsupported GIGO
	return resourceString
}

func buildFilterRequireObjectsIOwn(tx *sqlx.Tx, user models.ODUser) string {
	return fmt.Sprintf(" and o.ownedbyid = %d", getACMValueFor(tx, models.AACFlatten(user.DistinguishedName)))
}

func buildFilterExcludeObjectsIOrMyGroupsOwn(tx *sqlx.Tx, user models.ODUser) string {
	return " and o.ownedbyid not in (-1," + strings.Join(getACMValuesForUser(tx, user, "f_share"), ",") + ")"
}

func getACMValueFor(tx *sqlx.Tx, valueName string) int64 {
	var result []int64
	ret := int64(-1)
	sql := "select id from acmvalue2 where name = ?"
	err := tx.Select(&result, sql, valueName)
	if err != nil {
		log.Printf(err.Error())
	} else {
		if len(result) > 0 {
			ret = result[0]
		}
	}
	return ret
}

func getACMValuesForUser(tx *sqlx.Tx, user models.ODUser, keyName string) []string {
	sql := "select av.id from acmvalue2 av inner join useraocachepart uaocp on av.id = uaocp.uservalueid inner join acmkey2 ak on uaocp.userkeyid = ak.id and ak.name = ? "
	sql += "inner join user u on uaocp.userid = u.id and u.distinguishedname = ?"
	values := []string{}
	err := tx.Select(&values, sql, keyName, user.DistinguishedName)
	if err != nil {
		log.Printf("error getting acm values from key %s for user %s, %v", keyName, user.DistinguishedName, err)
		return values
	}
	return values
}

func getACMValueNamesForUser(tx *sqlx.Tx, user models.ODUser, keyName string) []string {
	sql := "select av.name from acmvalue2 av inner join useraocachepart uaocp on av.id = uaocp.uservalueid inner join acmkey2 ak on uaocp.userkeyid = ak.id and ak.name = ? "
	sql += "inner join user u on uaocp.userid = u.id and u.distinguishedname = ?"
	values := []string{}
	err := tx.Select(&values, sql, keyName, user.DistinguishedName)
	if err != nil {
		log.Printf("error getting acm values from key %s for user %s, %v", keyName, user.DistinguishedName, err)
		return values
	}
	return values
}

func buildFilterRequireObjectsIOrMyGroupsOwn(tx *sqlx.Tx, user models.ODUser) string {
	return " and o.ownedbyid in (-1," + strings.Join(getACMValuesForUser(tx, user, "f_share"), ",") + ")"
}

func buildListObjectsIOrMyGroupsOwn(tx *sqlx.Tx, user models.ODUser) []string {
	ownedby := []string{}
	ownedby = append(ownedby, "'user/"+MySQLSafeString2(user.DistinguishedName)+"'")
	ownedby = append(ownedby, buildListObjectsMyGroupOwns(tx, user)...)
	return ownedby
}

func buildListObjectsMyGroupOwns(tx *sqlx.Tx, user models.ODUser) []string {
	ownedby := []string{}
	groups := getGroupsFromSnippets(user)
	for _, group := range groups {
		if len(group) > 0 {
			if acmGrantee, err := getAcmGranteeInTransaction(tx, group); err == nil {
				resourceName := acmGrantee.ResourceNameRaw()
				ownedby = append(ownedby, "'"+MySQLSafeString2(resourceName)+"'")
			}
		}
	}
	return ownedby
}
func getGroupsFromSnippets(user models.ODUser) []string {
	if user.Snippets != nil {
		for _, rawFields := range user.Snippets.Snippets {
			switch rawFields.FieldName {
			case "f_share":
				return rawFields.Values
			default:
				continue
			}
		}
	}
	return []string{models.AACFlatten(user.DistinguishedName)}
}
