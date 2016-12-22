package server_test

import (
	"net/http"
	"testing"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"

	"encoding/json"
	"io/ioutil"

	cfg "decipher.com/object-drive-server/config"
)

func TestUserStats(t *testing.T) {
	clientID := 0
	if testing.Short() {
		t.Skip()
	}
	userStats := doUserStatsQuery(t)
	statsIntegrityCheck(t, userStats)

	//Create an object of known size, for its side-effects
	data := "0123456789"
	res, _ := doTestCreateObjectSimple(t, data, clientID, nil, nil, ValidAcmCreateObjectSimple)
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

	if userStats2.TotalObjects != userStats.TotalObjects+1 {
		t.Errorf("Expected to see one more object added")
	}

	diff := userStats2.TotalObjectsSize - userStats.TotalObjectsSize
	if int64(len(data)) != diff {
		t.Errorf("Expecting an %d byte object addition to add diff %d", len(data), diff)
	}
}

func doUserStatsQuery(t *testing.T) models.UserStats {
	// Get shares as the creator
	req, err := http.NewRequest("GET", "https://"+cfg.DockerVM+":"+cfg.Port+""+cfg.NginxRootURL+"/userstats", nil)
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
