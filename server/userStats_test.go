package server_test

import (
	"net/http"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"

	"encoding/json"
	"io/ioutil"
)

func TestUserStats(t *testing.T) {
	typeName := "TestUserStats"

	clientID := 0
	if testing.Short() {
		t.Skip()
	}
	userStats := doUserStatsQuery(t)
	statsIntegrityCheck(t, userStats)

	// See if type exists already
	typeObjects1 := 0
	typeSize1 := int64(0)
	for _, typeMetrics := range userStats.ObjectStorageMetrics {
		if typeMetrics.TypeName == typeName {
			typeObjects1 = typeMetrics.Objects
			typeSize1 = typeMetrics.ObjectsSize
		}
	}

	//Create an object of known size, for its side-effects
	data := "0123456789"
	res, _ := doTestCreateObjectSimpleWithType(t, data, clientID, nil, nil, ValidAcmCreateObjectSimple, typeName)
	if res == nil {
		t.Errorf("Unable to run query")
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("did not create new object")
		t.FailNow()
	}

	//Requery and diff
	userStats2 := doUserStatsQuery(t)
	statsIntegrityCheck(t, userStats)

	// Verify type is in the array
	typeFound2 := false
	typeObjects2 := 0
	typeSize2 := int64(0)
	for _, typeMetrics := range userStats2.ObjectStorageMetrics {
		if typeMetrics.TypeName == typeName {
			typeFound2 = true
			typeObjects2 = typeMetrics.Objects
			typeSize2 = typeMetrics.ObjectsSize
		}
	}
	if !typeFound2 {
		t.Errorf("No objects found of type %s", typeName)
	}
	if typeObjects2 != typeObjects1+1 {
		t.Errorf("Expected to see one more object added")
	}

	diff := typeSize2 - typeSize1
	if int64(len(data)) != diff {
		t.Errorf("Expecting an %d byte object addition to add diff %d", len(data), diff)
	}
}

func doUserStatsQuery(t *testing.T) models.UserStats {
	// Get shares as the creator
	req, err := http.NewRequest("GET", mountPoint+"/userstats", nil)
	if err != nil {
		t.Errorf("Unable to generate request:%v", err)
		t.FailNow()
	}
	clientid := 0
	res1, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res1.Body)
	if res1.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status %s for creator", res1.Status)
		t.FailNow()
	}
	var userStats models.UserStats
	userStats.TotalObjects = 42 // ensure that the test cannot trivially pass without writing the json
	bodyBytes, err := ioutil.ReadAll(res1.Body)
	if err != nil {
		t.Errorf("Could not marshal json:%s, %v", string(bodyBytes), err)
		t.FailNow()
	}
	err = json.Unmarshal(bodyBytes, &userStats)

	if err != nil {
		t.Errorf("Could not parse result:%v", err)
		t.FailNow()
	}
	return userStats
}

func statsIntegrityCheck(t *testing.T, userStats models.UserStats) {
	for i := range userStats.ObjectStorageMetrics {
		userStats.TotalObjects -= userStats.ObjectStorageMetrics[i].Objects
		userStats.TotalObjectsSize -= userStats.ObjectStorageMetrics[i].ObjectsSize
		userStats.TotalObjectsAndRevisions -= userStats.ObjectStorageMetrics[i].ObjectsAndRevisions
		userStats.TotalObjectsAndRevisionsSize -= userStats.ObjectStorageMetrics[i].ObjectsAndRevisionsSize
	}
	//Make sure that the totals add up
	if userStats.TotalObjects != 0 {
		t.Errorf("wrong totalObject count")
	}
	if userStats.TotalObjectsAndRevisions != 0 {
		t.Errorf("wrong totalObjectsAndRevisions count")
	}
	if userStats.TotalObjectsAndRevisionsSize != 0 {
		t.Errorf("wrong totalObjectsAndRevisionsSize count")
	}
	if userStats.TotalObjectsSize != 0 {
		t.Errorf("wrong totalObjectsSize count")
	}
}
