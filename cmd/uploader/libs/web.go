package libs

import (
	"crypto/tls"
	"decipher.com/oduploader/config"
	"decipher.com/oduploader/performance"
	"net/http"
	//"time"
	//"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strconv"
	"strings"
)

// Uploader is the state that the server references
type Uploader struct {
	Environment *config.Environment
	TLS         *tls.Config
	Tracker     *performance.JobReporters
}

// CreateUploader constructs an uploader object
func CreateUploader(env *config.Environment, tls *tls.Config) *Uploader {
	u := &Uploader{
		Environment: env,
		TLS:         tls,
	}
	deleteFromCache := func(name string) {
		log.Printf("can delete from cache: %s", name)
	}
	u.Tracker = performance.NewJobReporters(1024, deleteFromCache)
	return u
}

// CreateUploadServer generates an http server with the Uploader as handler
func (u Uploader) CreateUploadServer() *http.Server {
	addr :=
		u.Environment.TCPBind +
			":" +
			strconv.Itoa(u.Environment.TCPPort)

	return &http.Server{
		Addr:      addr,
		Handler:   u,
		TLSConfig: u.TLS,
		/*
			ReadTimeout:    10000 * time.Second, //This breaks big downloads
			WriteTimeout:   10000 * time.Second,
			MaxHeaderBytes: 1 << 20, //This prevents clients from DOS'ing us
		*/
	}
}

//sendErrorResponse will should be used to log the error codes sent
func (u Uploader) sendErrorResponse(w http.ResponseWriter, code int, err error, msg string) {
	log.Printf(msg+":%v", err)
	http.Error(w, msg, code)
}

func (u Uploader) statsRender(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "stats: %s", getIdentity(r))
}

/*
The local persistent cache and queues:

  ./outbound/   #queues
	  0/          #goroutine works to drain each numbered queue
		  d9e8e9d8.ciphertext
			d9e8e9d8.metadata
		1/
		2/
		3/
	./cache/
	    d9e8e9e8.ciphertext
			d9e8e9e8.metadata
*/

func (u Uploader) enqueue(part *multipart.Part) {
	//TODO: come up with a random name for the file
	//TODO: write ciphertext to the queue directory
	//TODO: write metadata for the ciphertext
	//TODO: write to goroutine to enquque for S3 upload later
	//TODO: the metadata existing is the signal that it must go to S3
	//TODO: the ciphertext without metadata means that it's orphaned.
}

func (u Uploader) upload(w http.ResponseWriter, r *http.Request) {
	beginAt := u.Tracker.BeginTime(performance.DownloadCounter, "")
	total := int64(0)
	multipartReader, err := r.MultipartReader()
	if err != nil {
		u.sendErrorResponse(w, 500, err, "failed to get a multipart reader")
	} else {
		stillReading := true
		for stillReading {
			part, err := multipartReader.NextPart()
			if err == io.EOF {
				//We are done reading the part
			} else {
				if err != nil {
					u.sendErrorResponse(w, 500, err, "error getting a part")
				} else {
					u.enqueue(part)
				}
			}
		}
	}
	u.Tracker.EndTime(performance.DownloadCounter, beginAt, "", performance.SizeJob(total))
}

func (u Uploader) download(w http.ResponseWriter, r *http.Request) {
	beginAt := u.Tracker.BeginTime(performance.DownloadCounter, "")
	total := int64(0)
	fmt.Fprintf(w, "download: %s", getIdentity(r))
	//TODO: lookup randomid from filename
	//  --- perhaps grant contains original filename, randomid of ciphertext
	u.Tracker.EndTime(performance.DownloadCounter, beginAt, "", performance.SizeJob(total))
}

// ServeHTTP is the muxer for this http server
func (u Uploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.RequestURI)
	//Implementations need to increment this - need a way to pass it around
	switch {
	case strings.Compare(r.RequestURI, "/stats") == 0:
		u.statsRender(w, r)
	case strings.Compare(r.RequestURI, "/upload") == 0:
		u.upload(w, r)
	case strings.Compare(r.RequestURI, "/download") == 0:
		u.download(w, r)
	case strings.Compare(r.RequestURI, "/") == 0:
		u.upload(w, r)
	default:
		w.WriteHeader(404)
	}
}

func getIdentity(r *http.Request) string {
	//TODO: otherwise, pull from headers
	if r.TLS != nil {
		if r.TLS.PeerCertificates != nil {
			if len(r.TLS.PeerCertificates) > 0 {
				return config.GetDNFromCert(r.TLS.PeerCertificates[0].Subject)
			}
		}
	}
	return "anonymous"
}
