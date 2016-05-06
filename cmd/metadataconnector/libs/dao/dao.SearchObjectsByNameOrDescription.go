package dao

import (
	"log"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"decipher.com/object-drive-server/protocol"
)

// SearchObjectsByNameOrDescription retrieves a list of Objects, their
// Permissions and optionally Properties in object drive that are
// available to the user making the call, matching any specified
// filter settings on the paging request, and ordered by sort settings
func (dao *DataAccessLayer) SearchObjectsByNameOrDescription(user models.ODUser, pagingRequest protocol.PagingRequest, loadProperties bool) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := searchObjectsByNameOrDescriptionInTransaction(tx, user, pagingRequest, loadProperties)
	if err != nil {
		log.Printf("Error in SearchObjectsByNameOrDescription: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func searchObjectsByNameOrDescriptionInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest, loadProperties bool) (models.ODObjectResultset, error) {
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
        ,o.isUSPersonsData
        ,o.isFOIAExempt        
        ,ot.name typeName     
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op	on o.id = op.objectid and op.isdeleted = 0 and op.allowread = 1
        inner join object_acm acm on o.id = acm.objectid            
    where 
        o.isdeleted = 0 
        and o.isexpunged = 0 
        and o.isancestordeleted = 0`
	query += buildFilterForUserACMShare(user)
	query += buildFilterForUserACM(user)
	query += buildFilterSortAndLimit(pagingRequest)

	//log.Println(query)
	err := tx.Select(&response.Objects, query, user.DistinguishedName)
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
func buildOrderBy(pagingRequest protocol.PagingRequest) string {
	out := ` order by `
	if len(pagingRequest.SortSettings) > 0 {
		for _, sortSetting := range pagingRequest.SortSettings {
			switch strings.ToLower(sortSetting.SortField) {
			case "changecount", "version":
				out += ` o.changecount`
			case "createdby", "creator":
				out += ` o.createdby`
			case "createddate":
				out += ` o.createddate`
			case "contentsize", "size":
				out += ` o.contentSize`
			case "contenttype", "mimetype":
				out += ` o.contenttype`
			case "description", "abstract":
				out += ` o.description`
			case "id":
				out += ` o.id`
			case "modifiedby", "modifier":
				out += ` o.modifiedby`
			case "modifieddate", "date":
				out += ` o.modifieddate`
			case "name", "title":
				out += ` o.name`
			case "ownedby", "owner":
				out += ` o.ownedby`
			case "typename", "type", "kind":
				out += ` ot.name`
			default:
				// unrecognized/unhandled field
				continue
			}
			if sortSetting.SortAscending {
				out += ` asc,`
			} else {
				out += ` desc,`
			}
		}
		if strings.HasSuffix(out, ",") {
			out = strings.TrimRight(out, ",")
		}
	} else {
		// By default, sort by the modified date, newest first
		out += ` o.modifieddate desc`
	}
	return out
}

// buildOrderByArchive first builds up the order by clause from paging request,
// and then converts to reference the archive table
func buildOrderByArchive(pagingRequest protocol.PagingRequest) string {
	a := buildOrderBy(pagingRequest)
	a = strings.Replace(a, " o.", " ao.", -1)
	return a
}

func buildFilter(pagingRequest protocol.PagingRequest) string {
	out := ``
	if len(pagingRequest.FilterSettings) > 0 {
		for _, filterSetting := range pagingRequest.FilterSettings {
			switch strings.ToLower(filterSetting.FilterField) {
			case "changecount", "version":
				out += ` or o.changecount`
			case "createdby", "creator":
				out += ` or o.createdby`
			case "createddate":
				out += ` or o.createddate`
			case "contentsize", "size":
				out += ` or o.contentSize`
			case "contenttype", "mimetype":
				out += ` or o.contenttype`
			case "description", "abstract":
				out += ` or o.description`
			case "id":
				out += ` or o.id`
			case "modifiedby", "modifier":
				out += ` or o.modifiedby`
			case "modifieddate", "date":
				out += ` or o.modifieddate`
			case "name", "title":
				out += ` or o.name`
			case "ownedby", "owner":
				out += ` or o.ownedby`
			case "typename", "type", "kind":
				out += ` or ot.name`
			default:
				// unrecognized/unhandled field
				continue
			}

			// TODO: Security cleanse this tightly.
			switch strings.ToLower(filterSetting.Condition) {
			case "contains":
				out += ` like '%` + MySQLSafeString(filterSetting.Expression) + `%'`
			default: // "equals":
				out += ` = '` + MySQLSafeString(filterSetting.Expression) + `'`
			}
		}
		out += `)`
		out = strings.Replace(out, " or ", "(", 1)
	} else {
		out = "(1=1)"
	}
	return out
}
func buildFilterArchive(pagingRequest protocol.PagingRequest) string {
	a := buildFilter(pagingRequest)
	a = strings.Replace(a, " o.", " ao.", -1)
	return a
}
func buildFilterSortAndLimit(pagingRequest protocol.PagingRequest) string {
	limit := GetLimit(pagingRequest.PageNumber, pagingRequest.PageSize)
	offset := GetOffset(pagingRequest.PageNumber, pagingRequest.PageSize)
	sqlStatementSuffix := ``
	sqlStatementSuffix += ` and ` + buildFilter(pagingRequest)
	sqlStatementSuffix += buildOrderBy(pagingRequest)
	sqlStatementSuffix += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	return sqlStatementSuffix
}
func buildFilterSortAndLimitArchive(pagingRequest protocol.PagingRequest) string {
	a := buildFilterSortAndLimit(pagingRequest)
	a = strings.Replace(a, " o.", " ao.", -1)
	return a
}

func buildFilterForUserACMShare(user models.ODUser) string {

	// Return if user.Snippets not defined, or there were no shares.
	defaultSQL := " and op.grantee = ? "

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
		sql = " and (op.grantee = ? "
		for _, shareValue := range shareSnippet.Values {
			if !strings.Contains(shareValue, "cusou") && !strings.Contains(shareValue, "governmentcus") {
				sql += " or op.grantee = '" + MySQLSafeString(shareValue) + "'"
			}
		}
		sql += ") "
	} else {
		sql = defaultSQL
	}

	return sql
}

func buildFilterForUserACM(user models.ODUser) string {

	if user.Snippets == nil {
		return " "
	}

	// The user object passed in has the AAC Snippets on it already.

	// sql is going to be the returned portion of the where clause built up from the snippets
	var sql string
	// Now iterate all the fields building up the where clause portion
	for _, rawFields := range user.Snippets.Snippets {
		fieldName := "acm."
		switch rawFields.FieldName {
		case "f_clearance", "f_oc_org", "f_missions", "f_regions", "f_macs", "f_sci_ctrls", "f_accms", "f_sar_id", "f_atom_energy", "f_dissem_countries":
			fieldName += rawFields.FieldName
		case "dissem_countries":
			fieldName += "f_dissem_countries"
		default:
			// All other snippet fields not used here (f_share has its own handler)
			continue
		}
		switch rawFields.Treatment {
		case "disallow":
			if len(rawFields.Values) > 0 {
				sql += " and (" + fieldName + " is null or " + fieldName + " = '' or " + fieldName + " = ',,' or (1=1"
				for _, value := range rawFields.Values {
					sql += " and " + fieldName + " not like '%," + MySQLSafeString(value) + ",%'"
				}
				sql += ")) "
			}
		case "allowed":
			sql += " and (" + fieldName + " is null or " + fieldName + " = '' or " + fieldName + " = ',,'"
			for _, value := range rawFields.Values {
				sql += " or " + fieldName + " like '%," + MySQLSafeString(value) + ",%'"
			}
			sql += ") "
		default:
			// Treatment type not handled. Log it and continue
			log.Printf("Unhandled treatment type when assembling ACM filter: %s", rawFields.Treatment)
			continue
		}
	}

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
