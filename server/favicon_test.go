package server_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	cfg "decipher.com/object-drive-server/config"
)

func TestFaviconDefault(t *testing.T) {

	s := NewFakeServerWithDAOUsers()
	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	s.AclImpersonationWhitelist = append(s.AclImpersonationWhitelist, whitelistedDN)
	// This is the equivalent of the default for actual server since this test is in libs/server
	s.StaticDir = filepath.Join("static")

	r, err := http.NewRequest("GET", cfg.RootURL+"/favicon.ico", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Add("USER_DN", fakeDN1)
	r.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	r.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected OK, got %v", w.Code)
	}

	if w.Body.Len() != 318 {
		t.Errorf("Icon file for favicon.ico was not expected size %d. It is reported as %d", 318, w.Body.Len())
	}
}

func TestFaviconFailsForNoStaticDir(t *testing.T) {

	s := NewFakeServerWithDAOUsers()
	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	s.AclImpersonationWhitelist = append(s.AclImpersonationWhitelist, whitelistedDN)
	// Simulates staticDir "" for server startup
	s.StaticDir = ""

	r, err := http.NewRequest("GET", cfg.RootURL+"/favicon.ico", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Add("USER_DN", fakeDN1)
	r.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	r.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected OK, got %v", w.Code)
	}

}
