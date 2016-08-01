package server_test

import (
	"strings"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/protocol"
)

func TestRemoveObjectShareFromCaller(t *testing.T) {

	t.Logf("Create as tester10, RUS to tester01, R to odrive_g2")
	creator := 0
	permissions := []protocol.Permission{makePermission(fakeDN0, true, true, true, true, true), makePermission(fakeDN1, false, true, true, false, true)}
	acmShare := `"share": {` + makeGroupShareString("DCTC", "DCTC", "ODrive_G2") + `}`
	newObject := createSharedObjectForTestRemoveObjectShare(t, creator, acmShare, permissions)

	t.Logf("Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 2, 3, 4, 5, 0)
	shouldNotHaveReadForObjectID(t, newObject.ID, 6, 7, 8, 9)

	t.Logf("Remove tester01 Shares to as tester01")
	delegate := 1
	uriRemoveShare := host + cfg.NginxRootURL + "/shared/" + newObject.ID
	removeShareRequest := protocol.ObjectShare{}
	removeShareRequest.Share = makeUserShare(fakeDN1)
	removeShareReq := makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err := clients[delegate].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, removeShareRes, "Bad status when removing share from object")
	var updatedObject protocol.Object
	err = util.FullDecode(removeShareRes.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs")
	t.Logf("tester1 retains read access from ODrive_G2, but will low update + share")
	shouldHaveReadForObjectID(t, updatedObject.ID, 1, 2, 3, 4, 5, 0)
	shouldNotHaveReadForObjectID(t, updatedObject.ID, 6, 7, 8, 9)

	t.Logf("Attempt to Remove Shares to tester1 again")
	removeShareReq = makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err = clients[delegate].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 403, removeShareRes, "Bad status when removing share from object")
	util.FinishBody(removeShareRes.Body)

}
func TestRemoveObjectShareFromOtherUser(t *testing.T) {
	t.Logf("Create as tester10, RUS to tester01, R to odrive_g1")
	creator := 0
	permissions := []protocol.Permission{makePermission(fakeDN0, true, true, true, true, true), makePermission(fakeDN1, false, true, true, false, true)}
	acmShare := `"share": {` + makeGroupShareString("DCTC", "DCTC", "ODrive_G1") + `}`
	newObject := createSharedObjectForTestRemoveObjectShare(t, creator, acmShare, permissions)

	t.Logf("Verify tester 1, 6-10 can read it, but not 2-4 or other certs")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 6, 7, 8, 9, 0)
	shouldNotHaveReadForObjectID(t, newObject.ID, 2, 3, 4, 5)

	t.Logf("Remove tester01 Shares to as tester10")
	uriRemoveShare := host + cfg.NginxRootURL + "/shared/" + newObject.ID
	removeShareRequest := protocol.ObjectShare{}
	removeShareRequest.Share = makeUserShare(fakeDN1)
	removeShareReq := makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err := clients[creator].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, removeShareRes, "Bad status when removing share from object")
	var updatedObject protocol.Object
	err = util.FullDecode(removeShareRes.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("Verify tester 6-10 can read it, but not 1-5 or other certs")
	shouldHaveReadForObjectID(t, updatedObject.ID, 6, 7, 8, 9, 0)
	shouldNotHaveReadForObjectID(t, updatedObject.ID, 1, 2, 3, 4, 5)
}
func TestRemoveObjectShareFromOwner(t *testing.T) {
	t.Logf("Create as tester10, RUS to tester01, R to odrive_g2")
	creator := 0
	permissions := []protocol.Permission{makePermission(fakeDN0, true, true, true, true, true), makePermission(fakeDN1, false, true, true, false, true)}
	acmShare := `"share": {` + makeGroupShareString("DCTC", "DCTC", "ODrive_G2") + `}`
	newObject := createSharedObjectForTestRemoveObjectShare(t, creator, acmShare, permissions)

	t.Logf("Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 2, 3, 4, 5, 0)
	shouldNotHaveReadForObjectID(t, newObject.ID, 6, 7, 8, 9)

	t.Logf("As Tester01 Remove Shares to tester10")
	delegate := 1
	uriRemoveShare := host + cfg.NginxRootURL + "/shared/" + newObject.ID
	removeShareRequest := protocol.ObjectShare{}
	removeShareRequest.Share = makeUserShare(fakeDN0)
	removeShareReq := makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err := clients[delegate].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 403, removeShareRes, "Bad status when removing share from object")
	util.FinishBody(removeShareRes.Body)
}
func TestRemoveObjectShareFromNonExistentUser(t *testing.T) {
	t.Logf("Create as tester10, RUS to tester01, R to odrive_g2")
	creator := 0
	permissions := []protocol.Permission{makePermission(fakeDN0, true, true, true, true, true), makePermission(fakeDN1, false, true, true, false, true)}
	acmShare := `"share": {` + makeGroupShareString("DCTC", "DCTC", "ODrive_G2") + `}`
	newObject := createSharedObjectForTestRemoveObjectShare(t, creator, acmShare, permissions)

	t.Logf("Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 2, 3, 4, 5, 0)
	shouldNotHaveReadForObjectID(t, newObject.ID, 6, 7, 8, 9)

	t.Logf("As Tester01 Remove Shares to nonexistentuser")
	delegate := 1
	uriRemoveShare := host + cfg.NginxRootURL + "/shared/" + newObject.ID
	removeShareRequest := protocol.ObjectShare{}
	removeShareRequest.Share = makeUserShare("nonexistentuser")
	removeShareReq := makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err := clients[delegate].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, removeShareRes, "Bad status when removing share from object")
	var updatedObject protocol.Object
	err = util.FullDecode(removeShareRes.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs")
	t.Logf("tester1 retains read access from ODrive_G2, but will low update + share")
	shouldHaveReadForObjectID(t, updatedObject.ID, 1, 2, 3, 4, 5, 0)
	shouldNotHaveReadForObjectID(t, updatedObject.ID, 6, 7, 8, 9)
}
func TestRemoveObjectShareFromCallerGroup(t *testing.T) {
	t.Logf("Create as tester10, R to odrive_g1, R to odrive_g2, RUS to tester01")
	creator := 0
	permissions := []protocol.Permission{makePermission(fakeDN0, true, true, true, true, true), makePermission(fakeDN1, false, true, true, false, true)}
	acmShare := `"share": {` + makeGroupShareString("DCTC", "DCTC", `ODrive_G1","ODrive_G2`) + `}`
	newObject := createSharedObjectForTestRemoveObjectShare(t, creator, acmShare, permissions)

	t.Logf("Verify tester 1-0 can read it")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0)

	t.Logf("As Tester01 Remove Shares to odrive_g2")
	delegate := 1
	uriRemoveShare := host + cfg.NginxRootURL + "/shared/" + newObject.ID
	removeShareRequest := protocol.ObjectShare{}
	removeShareRequest.Share = makeGroupShare("DCTC", "DCTC", "ODrive_G2")
	removeShareReq := makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err := clients[delegate].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, removeShareRes, "Bad status when removing share from object")
	var updatedObject protocol.Object
	err = util.FullDecode(removeShareRes.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("Verify tester 1, 6-10 can read it, but not 2-5")
	shouldHaveReadForObjectID(t, updatedObject.ID, 1, 6, 7, 8, 9, 0)
	shouldNotHaveReadForObjectID(t, updatedObject.ID, 2, 3, 4, 5)
}
func TestRemoveObjectShareFromOtherGroup(t *testing.T) {
	t.Logf("Create as tester10, R to odrive_g1, R to odrive_g2, RUS to tester01")
	creator := 0
	permissions := []protocol.Permission{makePermission(fakeDN0, true, true, true, true, true), makePermission(fakeDN1, false, false, true, false, true)}
	acmShare := `"share": {` + makeGroupShareString("DCTC", "DCTC", `ODrive_G1","ODrive_G2`) + `}`
	newObject := createSharedObjectForTestRemoveObjectShare(t, creator, acmShare, permissions)

	t.Logf("Verify tester 1-0 can read it")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0)

	t.Logf("As Tester01 Remove Shares to odrive_g1")
	delegate := 1
	uriRemoveShare := host + cfg.NginxRootURL + "/shared/" + newObject.ID
	removeShareRequest := protocol.ObjectShare{}
	removeShareRequest.Share = makeGroupShare("DCTC", "DCTC", "ODrive_G1")
	removeShareReq := makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err := clients[delegate].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, removeShareRes, "Bad status when removing share from object")
	var updatedObject protocol.Object
	err = util.FullDecode(removeShareRes.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("Verify tester 1-5 and 10 can read it, but not 6-9")
	shouldHaveReadForObjectID(t, updatedObject.ID, 1, 2, 3, 4, 5, 0)
	shouldNotHaveReadForObjectID(t, updatedObject.ID, 6, 7, 8, 9)
}
func TestRemoveObjectShareFromEveryoneGroup(t *testing.T) {
	t.Logf("Create as tester10, no special perms")
	creator := 0
	permissions := []protocol.Permission{}
	acmShare := ""
	newObject := createSharedObjectForTestRemoveObjectShare(t, creator, acmShare, permissions)

	t.Logf("Verify tester 1-0 can read it from everyone group")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0)

	t.Logf("As Tester10 Remove Shares to everyone group")
	uriRemoveShare := host + cfg.NginxRootURL + "/shared/" + newObject.ID
	removeShareRequest := protocol.ObjectShare{}
	removeShareRequest.Share = makeUserShare(models.EveryoneGroup)
	removeShareReq := makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err := clients[creator].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, removeShareRes, "Bad status when removing share from object")
	var updatedObject protocol.Object
	err = util.FullDecode(removeShareRes.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("Verify tester 10 can read it, but not 1-9")
	shouldHaveReadForObjectID(t, updatedObject.ID, 0)
	shouldNotHaveReadForObjectID(t, updatedObject.ID, 1, 2, 3, 4, 5, 6, 7, 8, 9)

}
func TestRemoveObjectShareWithoutPermission(t *testing.T) {
	t.Logf("Create as tester10, R to everyone")
	creator := 0
	permissions := []protocol.Permission{}
	acmShare := ""
	newObject := createSharedObjectForTestRemoveObjectShare(t, creator, acmShare, permissions)

	t.Logf("Verify tester 1-0 can read it")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0)

	t.Logf("As Tester01 Remove Shares to Everyone")
	delegate := 1
	uriRemoveShare := host + cfg.NginxRootURL + "/shared/" + newObject.ID
	removeShareRequest := protocol.ObjectShare{}
	removeShareRequest.Share = makeUserShare(models.EveryoneGroup)
	removeShareReq := makeHTTPRequestFromInterface(t, "DELETE", uriRemoveShare, removeShareRequest)
	removeShareRes, err := clients[delegate].Client.Do(removeShareReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 403, removeShareRes, "Bad status when removing share from object")
	util.FinishBody(removeShareRes.Body)

	t.Logf("Verify tester 1-0 can still read it")
	shouldHaveReadForObjectID(t, newObject.ID, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0)
}

func makePermission(grantee string, allowCreate bool, allowRead bool, allowUpdate bool, allowDelete bool, allowShare bool) protocol.Permission {
	permission := protocol.Permission{
		Grantee:     grantee,
		AllowCreate: allowCreate,
		AllowRead:   allowRead,
		AllowUpdate: allowUpdate,
		AllowDelete: allowDelete,
		AllowShare:  allowShare}
	return permission
}

func createSharedObjectForTestRemoveObjectShare(t *testing.T, clientid int, acmShare string, permissions []protocol.Permission) protocol.Object {

	// ### Create object as the client
	t.Logf("Creating object with shares for TestRemoveObjectShare as %d", clientid)
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestRemoveObjectShare"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.ContentSize = 0
	// default share read to everyone
	acm := `{"version":"2.1.0","classif":"U","share":{}}`
	if len(acmShare) > 0 {
		acm = strings.Replace(acm, `"share":{}`, acmShare, -1)
	}
	createObjectRequest.RawAcm = models.ToNullString(acm)
	// permissions if any passed in
	createObjectRequest.Permissions = permissions
	// http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	createReq := makeHTTPRequestFromInterface(t, "POST", uriCreate, createObjectRequest)
	// exec and get response
	createRes, err := clients[clientid].Client.Do(createReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")
	return createdObject
}
