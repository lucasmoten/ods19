package dao_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/server"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

func TestDAOSearchObjectsByNameOrDescription(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	guid, _ := util.NewGUID()
	timeSuffix := strconv.FormatInt(time.Now().Unix(), 10) + guid

	// Create object 1 and maintain reference to delete later
	obj1 := setupObjectForDAOSearchObjectsTest("Search Object 1" + timeSuffix)
	dbObject1, err := d.CreateObject(&obj1)
	if err != nil {
		t.Error(err)
	}
	if dbObject1.ID == nil {
		t.Error("expected ID to be set")
	}

	// Create object 2 and maintain reference to delete later
	obj2 := setupObjectForDAOSearchObjectsTest("Search Object 2" + timeSuffix)
	dbObject2, err := d.CreateObject(&obj2)
	if err != nil {
		t.Error(err)
	}
	if dbObject2.ID == nil {
		t.Error("expected ID to be set")
	}

	pagingRequest := dao.PagingRequest{}

	// Search for objects that have name or description with phrase 'Search Object 1'
	filterNameAsSearch1 := dao.FilterSetting{}
	filterNameAsSearch1.FilterField = "name"
	filterNameAsSearch1.Condition = "contains"
	filterNameAsSearch1.Expression = "Search Object 1" + timeSuffix
	filterDescriptionAsSearch1 := dao.FilterSetting{}
	filterDescriptionAsSearch1.FilterField = "description"
	filterDescriptionAsSearch1.Condition = "contains"
	filterDescriptionAsSearch1.Expression = "Search Object 1" + timeSuffix
	pagingRequest.FilterSettings = make([]dao.FilterSetting, 0)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterNameAsSearch1)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterDescriptionAsSearch1)
	searchResults1, err := d.SearchObjectsByNameOrDescription(users[1], pagingRequest, false)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if searchResults1.TotalRows != 1 {
		t.Logf("expected 1 result searching for Search Object 1, got %d for %s", searchResults1.TotalRows, timeSuffix)
		t.Fail()
	}

	// Search for objects that have name or description with phrase 'Search Object 2'
	filterNameAsSearch2 := dao.FilterSetting{}
	filterNameAsSearch2.FilterField = "name"
	filterNameAsSearch2.Condition = "contains"
	filterNameAsSearch2.Expression = "Search Object 2" + timeSuffix
	filterDescriptionAsSearch2 := dao.FilterSetting{}
	filterDescriptionAsSearch2.FilterField = "description"
	filterDescriptionAsSearch2.Condition = "contains"
	filterDescriptionAsSearch2.Expression = "Search Object 2" + timeSuffix
	pagingRequest.FilterSettings = make([]dao.FilterSetting, 0)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterNameAsSearch2)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterDescriptionAsSearch2)
	searchResults2, err := d.SearchObjectsByNameOrDescription(users[1], pagingRequest, false)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if searchResults2.TotalRows != 1 {
		t.Logf("expected 1 result searching for Search Object 2, got %d for %s", searchResults2.TotalRows, timeSuffix)
		t.Fail()
	}

	// Search for objects that have name or description with phrase 'Search'
	filterNameAsSearch3 := dao.FilterSetting{}
	filterNameAsSearch3.FilterField = "name"
	filterNameAsSearch3.Condition = "contains"
	filterNameAsSearch3.Expression = timeSuffix
	filterDescriptionAsSearch3 := dao.FilterSetting{}
	filterDescriptionAsSearch3.FilterField = "description"
	filterDescriptionAsSearch3.Condition = "contains"
	filterDescriptionAsSearch3.Expression = timeSuffix
	pagingRequest.FilterSettings = make([]dao.FilterSetting, 0)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterNameAsSearch3)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterDescriptionAsSearch3)
	searchResults3, err := d.SearchObjectsByNameOrDescription(users[1], pagingRequest, false)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if searchResults3.TotalRows != 2 {
		t.Logf("expected 2 results from searching, got %d for %s", searchResults3.TotalRows, timeSuffix)
		t.Fail()
	}

	// Attempt some sql injection
	filterNameAsSearch4 := dao.FilterSetting{}
	filterNameAsSearch4.FilterField = "name"
	filterNameAsSearch4.Condition = "contains"
	filterNameAsSearch4.Expression = "Search Object 3" // intentionally wont match anything
	filterDescriptionAsSearch4 := dao.FilterSetting{}
	filterDescriptionAsSearch4.FilterField = "description"
	filterDescriptionAsSearch4.Condition = "equals"
	filterDescriptionAsSearch4.Expression = "\\') or (o.name = % \\)) /* " // intends to break or include all objects
	pagingRequest.FilterSettings = make([]dao.FilterSetting, 0)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterNameAsSearch4)
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, filterDescriptionAsSearch4)
	searchResults4, err := d.SearchObjectsByNameOrDescription(users[1], pagingRequest, false)
	if err != nil {
		t.Error(err)
	}
	if searchResults4.TotalRows != 0 {
		t.Error("expected 0 results from searching")
	}
}

func TestDAOSearchObjectsAndOrFilter(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	guid, _ := util.NewGUID()
	timeSuffix := strconv.FormatInt(time.Now().Unix(), 10) + guid
	baseName := "AndOrFilter" + timeSuffix
	obj1Name := baseName + " Object 1"
	obj2Name := baseName + " Object 2"

	// Create object 1 and maintain reference to delete later
	obj1 := setupObjectForDAOSearchObjectsTest(obj1Name)
	d.CreateObject(&obj1)

	// Create object 2 and maintain reference to delete later
	obj2 := setupObjectForDAOSearchObjectsTest(obj2Name)
	d.CreateObject(&obj2)

	// OR test (default filter match type)
	pagingRequestOR := dao.PagingRequest{FilterSettings: []dao.FilterSetting{
		dao.FilterSetting{FilterField: "name", Condition: "contains", Expression: obj1Name},
		dao.FilterSetting{FilterField: "name", Condition: "contains", Expression: obj2Name}}}
	searchResultsOR, err := d.SearchObjectsByNameOrDescription(users[1], pagingRequestOR, false)
	if err != nil {
		t.Error(err)
	}
	if searchResultsOR.TotalRows != 2 {
		t.Logf("expected 2 results from searching, got %d", searchResultsOR.TotalRows)
		t.Fail()
	}

	// AND test (match type set)
	pagingRequestAND := dao.PagingRequest{FilterSettings: []dao.FilterSetting{
		dao.FilterSetting{FilterField: "name", Condition: "contains", Expression: obj1Name},
		dao.FilterSetting{FilterField: "owner", Condition: "contains", Expression: usernames[1]}},
		FilterMatchType: "and"}
	searchResultsAND, err := d.SearchObjectsByNameOrDescription(users[1], pagingRequestAND, false)
	if err != nil {
		t.Error(err)
	}
	if searchResultsAND.TotalRows != 1 {
		t.Logf("expected 1 results from searching, got %d", searchResultsAND.TotalRows)
		t.Fail()
	}

}

func setupObjectForDAOSearchObjectsTest(name string) models.ODObject {
	var obj models.ODObject
	obj.Name = "Test " + name + " Name"
	obj.Description = models.ToNullString(name + " Description")
	obj.CreatedBy = usernames[1]
	obj.TypeName = models.ToNullString("File")
	permissions := make([]models.ODObjectPermission, 1)
	permissions[0].Grantee = models.AACFlatten(obj.CreatedBy)
	permissions[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions[0].CreatedBy)
	permissions[0].AcmGrantee.Grantee = permissions[0].Grantee
	permissions[0].AcmGrantee.ResourceString = models.ToNullString("user/" + obj.CreatedBy)
	permissions[0].AcmGrantee.UserDistinguishedName = models.ToNullString(permissions[0].CreatedBy)
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	permissions[0].AllowShare = true
	obj.Permissions = permissions
	obj.RawAcm.String = server.ValidACMUnclassified
	return obj
}
