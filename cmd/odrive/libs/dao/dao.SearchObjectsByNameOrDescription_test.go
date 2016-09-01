package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOSearchObjectsByNameOrDescription(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	// Create object 1 and maintain reference to delete later
	obj1 := setupObjectForDAOSearchObjectsTest("Search Object 1")
	dbObject1, err := d.CreateObject(&obj1)
	if err != nil {
		t.Error(err)
	}
	if dbObject1.ID == nil {
		t.Error("expected ID to be set")
	}

	// Create object 2 and maintain reference to delete later
	obj2 := setupObjectForDAOSearchObjectsTest("Search Object 2")
	dbObject2, err := d.CreateObject(&obj2)
	if err != nil {
		t.Error(err)
	}
	if dbObject2.ID == nil {
		t.Error("expected ID to be set")
	}

	pagingRequest := protocol.PagingRequest{}

	// Search for objects that have name or description with phrase 'Search Object 1'
	filterNameAsSearch1 := protocol.FilterSetting{}
	filterNameAsSearch1.FilterField = "name"
	filterNameAsSearch1.Condition = "contains"
	filterNameAsSearch1.Expression = "Search Object 1"
	filterDescriptionAsSearch1 := protocol.FilterSetting{}
	filterDescriptionAsSearch1.FilterField = "description"
	filterDescriptionAsSearch1.Condition = "contains"
	filterDescriptionAsSearch1.Expression = "Search Object 1"
	pagingRequest.FilterSettings = make([]protocol.FilterSetting, 0)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterNameAsSearch1)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterDescriptionAsSearch1)
	user := setupUserWithSnippets(usernames[1])
	searchResults1, err := d.SearchObjectsByNameOrDescription(user, pagingRequest, false)
	if err != nil {
		t.Error(err)
	}
	if searchResults1.TotalRows != 1 {
		t.Error("expected 1 result searching for Search Object 1")
	}

	// Search for objects that have name or description with phrase 'Search Object 2'
	filterNameAsSearch2 := protocol.FilterSetting{}
	filterNameAsSearch2.FilterField = "name"
	filterNameAsSearch2.Condition = "contains"
	filterNameAsSearch2.Expression = "Search Object 2"
	filterDescriptionAsSearch2 := protocol.FilterSetting{}
	filterDescriptionAsSearch2.FilterField = "description"
	filterDescriptionAsSearch2.Condition = "contains"
	filterDescriptionAsSearch2.Expression = "Search Object 2"
	pagingRequest.FilterSettings = make([]protocol.FilterSetting, 0)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterNameAsSearch2)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterDescriptionAsSearch2)
	searchResults2, err := d.SearchObjectsByNameOrDescription(user, pagingRequest, false)
	if err != nil {
		t.Error(err)
	}
	if searchResults2.TotalRows != 1 {
		t.Error("expected 1 result searching for Search Object 2")
	}

	// Search for objects that have name or description with phrase 'Search'
	filterNameAsSearch3 := protocol.FilterSetting{}
	filterNameAsSearch3.FilterField = "name"
	filterNameAsSearch3.Condition = "contains"
	filterNameAsSearch3.Expression = "Search"
	filterDescriptionAsSearch3 := protocol.FilterSetting{}
	filterDescriptionAsSearch3.FilterField = "description"
	filterDescriptionAsSearch3.Condition = "contains"
	filterDescriptionAsSearch3.Expression = "Search"
	pagingRequest.FilterSettings = make([]protocol.FilterSetting, 0)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterNameAsSearch3)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterDescriptionAsSearch3)
	searchResults3, err := d.SearchObjectsByNameOrDescription(user, pagingRequest, false)
	if err != nil {
		t.Error(err)
	}
	if searchResults3.TotalRows != 2 {
		t.Error("expected 3 results from searching")
	}

	// Attempt some sql injection
	filterNameAsSearch4 := protocol.FilterSetting{}
	filterNameAsSearch4.FilterField = "name"
	filterNameAsSearch4.Condition = "contains"
	filterNameAsSearch4.Expression = "Search Object 3" // intentionally wont match anything
	filterDescriptionAsSearch4 := protocol.FilterSetting{}
	filterDescriptionAsSearch4.FilterField = "description"
	filterDescriptionAsSearch4.Condition = "equals"
	filterDescriptionAsSearch4.Expression = "\\') or (o.name = % \\)) /* " // intends to break or include all objects
	pagingRequest.FilterSettings = make([]protocol.FilterSetting, 0)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterNameAsSearch4)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterDescriptionAsSearch4)
	searchResults4, err := d.SearchObjectsByNameOrDescription(user, pagingRequest, false)
	if err != nil {
		t.Error(err)
	}
	if searchResults4.TotalRows != 0 {
		t.Error("expected 0 results from searching")
	}

	// cleanup / delete the objects

	err = d.DeleteObject(user, dbObject1, true)
	if err != nil {
		t.Error(err)
	}
	err = d.DeleteObject(user, dbObject2, true)
	if err != nil {
		t.Error(err)
	}
}

func setupObjectForDAOSearchObjectsTest(name string) models.ODObject {
	var obj models.ODObject
	obj.Name = "Test " + name + " Name"
	obj.Description.String = name + " Description"
	obj.Description.Valid = true
	obj.CreatedBy = usernames[1]
	obj.TypeName.String = "File"
	obj.TypeName.Valid = true
	permissions := make([]models.ODObjectPermission, 1)
	permissions[0].Grantee = obj.CreatedBy
	permissions[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions[0].Grantee)
	permissions[0].AcmGrantee.Grantee = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.String = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.Valid = true
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	obj.Permissions = permissions
	obj.RawAcm.String = testhelpers.ValidACMUnclassified
	return obj
}

func setupUserWithSnippets(username string) models.ODUser {
	var user models.ODUser
	user.DistinguishedName = username

	snippet := acm.RawSnippetFields{}
	snippet.FieldName = "f_share"
	snippet.Treatment = "allowed"
	snippet.Values = make([]string, 1)
	snippet.Values[0] = username
	snippets := acm.ODriveRawSnippetFields{}
	snippets.Snippets = append(snippets.Snippets, snippet)
	user.Snippets = &snippets

	return user
}
