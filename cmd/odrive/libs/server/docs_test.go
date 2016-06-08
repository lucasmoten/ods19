package server_test

import (
	"fmt"
	"log"
	"net/http"
	"testing"

	cfg "decipher.com/object-drive-server/config"
)

func TestAPIDocs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// Setup a request to retrieve the API docs at root
	apiuri := host + cfg.NginxRootURL + "/"

	// do the request
	req, err := http.NewRequest("GET", apiuri, nil)
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}

}
