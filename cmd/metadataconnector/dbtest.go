package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
)

func main() {

	// Load Configuration from conf.json
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	serverConfig := appConfiguration.ServerSettings

	// Setup handle to the database
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	// Validate the DSN for the database by pinging it
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	// Setup web server
	s, err := makeServer(serverConfig, db)
	stls := serverConfig.GetTLSConfig()
	s.TLSConfig = &stls
	serverCertFile := serverConfig.ServerCertChain
	serverKeyFile := serverConfig.ServerKey
	//func ListenAndServeTLS(addr string, certFile string, keyFile string, handler Handler) error
	log.Println("Starting server on " + s.Addr)
	log.Fatalln(s.ListenAndServeTLS(serverCertFile, serverKeyFile))

	//dbtest()
}

/* ServeHTTP handles the routing of requests
 */
func (h MCServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.RequestURI() == "/":
		h.home(w, r)
	case r.URL.RequestURI() == "/listObjects":
		h.listObjects(w, r)
	case r.URL.RequestURI() == "/getObject":
		h.getObject(w, r)
	default:
		who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
		msg := who + " requested uri: " + r.URL.RequestURI() + " from address: " + r.RemoteAddr + " with user agent: " + r.UserAgent()
		log.Println("WARN: " + msg)
		http.Error(w, "Resource not found", 404)
	}
}

var pageTemplateStart = `
<html>
  <head><title>Object-Drive</title>
	<body>
		Method: %s
		<br />
		Distinguished Name:%s
		<br />
		<hr />
`

var pageTemplateEnd = `
	</body>
</html>
`

func (h MCServer) home(w http.ResponseWriter, r *http.Request) {
	who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
	r.Header.Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "Home Page", who)
	fmt.Fprintf(w, "Length of distinguished name: "+strconv.Itoa(len(who)))
	fmt.Fprintf(w, "<hr />")
	fmt.Fprintf(w, "<a href='/listObjects'>List All Objects at Root</a><br />")
	fmt.Fprintf(w, "<a href='/getObject'>Get Object</a><br />")
	fmt.Fprintf(w, pageTemplateEnd)
}
func (h MCServer) listObjects(w http.ResponseWriter, r *http.Request) {
	who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
	r.Header.Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "listObjects", who)

	response, err := dao.GetRootObjectsWithProperties(h.MetadataDB, "createdDate DESC", 1, 20)
	objects := response.Objects
	if err != nil {
		panic(err.Error())
	}
	fmt.Fprintf(w, "Page "+strconv.Itoa(response.PageNumber)+" of "+strconv.Itoa(response.PageCount)+".<br />")
	fmt.Fprintf(w, "Page Size: "+strconv.Itoa(response.PageSize)+", Page Rows: "+strconv.Itoa(response.PageRows)+", Total Rows: "+strconv.Itoa(response.TotalRows)+"<br />")
	fmt.Fprintf(w, `<table width="100%" border="1" bordercolor="red" style="width:100%;border:1px solid red;"><tr><td>ID</td><td>Created Date</td><td>Created By</td><td>Name</td></tr>`)
	for idx := range objects {
		object := objects[idx]

		fmt.Fprintf(w, "<tr><td>")
		fmt.Fprintf(w, hex.EncodeToString(object.ID))
		fmt.Fprintf(w, "</td><td>")
		fmt.Fprintf(w, object.CreatedDate.Format(time.RFC3339))
		fmt.Fprintf(w, "</td><td>")
		fmt.Fprintf(w, object.CreatedBy)
		fmt.Fprintf(w, "</td><td>")
		fmt.Fprintf(w, object.Name)
		fmt.Fprintf(w, "</td></tr>")
	}
	fmt.Fprintf(w, "</table>")
	fmt.Fprintf(w, pageTemplateEnd)
}
func (h MCServer) getObject(w http.ResponseWriter, r *http.Request) {
	who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
	r.Header.Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "getObject", who)
	fmt.Fprintf(w, pageTemplateEnd)
}
func (h MCServer) sendErrorResponse(w http.ResponseWriter, code int, err error, msg string) {
	log.Printf(msg+":%v", err)
	http.Error(w, msg, code)
}

/*
MCServer contains definition for the metadata server
*/
type MCServer struct {
	Port       int
	Bind       string
	Addr       string
	MetadataDB *sqlx.DB
}

func makeServer(serverConfig config.ServerSettingsConfiguration, db *sqlx.DB) (*http.Server, error) {
	h := MCServer{
		Port:       serverConfig.ListenPort,
		Bind:       serverConfig.ListenBind,
		Addr:       serverConfig.ListenBind + ":" + strconv.Itoa(serverConfig.ListenPort),
		MetadataDB: db,
	}
	return &http.Server{
		Addr:           string(h.Addr),
		Handler:        h,
		ReadTimeout:    10000 * time.Second, //This breaks big downloads
		WriteTimeout:   10000 * time.Second,
		MaxHeaderBytes: 1 << 20, //This prevents clients from DOS'ing us
	}, nil
}

func dbtest() {

	// Load Configuration from conf.json
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection

	// Setup handle to the database
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// Validate the DSN for the database by pinging it
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	// ===========================================================================
	// Retrieve Alice's root objects
	response, err := dao.GetRootObjectsWithPropertiesByOwner(db,
		"createdDate DESC", 1, 20, "Alice")
	objects := response.Objects
	if err != nil {
		panic(err.Error())
	}
	jsonData, err := json.MarshalIndent(response, "", "  ")
	jsonified := string(jsonData)
	fmt.Println(jsonified)
	// ===========================================================================
	// Choose a random object in the resultset
	rns := rand.NewSource(int64(time.Now().Nanosecond()))
	objectIndex := rand.New(rns).Intn(len(objects))
	// ===========================================================================
	// Add a new property to the chosen object
	fmt.Println("Adding property to " + strconv.Itoa(objectIndex))
	if len(objects) > objectIndex {
		newPropertyCreatedBy := objects[objectIndex].CreatedBy
		newPropertyName := "Prop" + strconv.Itoa(time.Now().Nanosecond())
		newPropertyValue := time.Now().Format(time.RFC3339)
		newPropertyClassification := "U"

		dao.AddPropertyToObject(db, newPropertyCreatedBy, objects[objectIndex].ID,
			newPropertyName, newPropertyValue, newPropertyClassification)
	}
	// ===========================================================================
	// Retrieve Alice's root objects
	response, err = dao.GetRootObjectsWithPropertiesByOwner(db,
		"createdDate DESC", 1, 20, "Alice")
	objects = response.Objects
	if err != nil {
		panic(err.Error())
	}
	jsonData, err = json.MarshalIndent(response, "", "  ")
	jsonified = string(jsonData)
	fmt.Println(jsonified)

}
