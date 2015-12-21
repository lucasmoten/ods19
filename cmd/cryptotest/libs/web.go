package libs

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

/* ServeHTTP handles the routing of requests
 */
func (h Uploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {

	//Upload a file into the bucket
	//      $HOMEBUCKET/?fileName
	case r.URL.RequestURI() == "/upload":
		{
			switch {
			case r.Method == "GET":
				h.serveHTTPUploadGET(w, r)
			case r.Method == "POST":
				h.serveHTTPUploadPOST(w, r)
			}
		}

	//Get a file from the bucket
	//      $HOMEBUCKET/:fileName
	//This will get a specific file for a user.  The slash is used
	//to distinguish between being an actual file, and just wanting
	//a listing so that the user doesn't need to remember what was
	//uploaded.
	case strings.HasPrefix(r.URL.RequestURI(), "/download/"):
		{
			h.serveHTTPDownloadGET(w, r)
		}

	//This will get a listing of files for this user
	case strings.HasPrefix(r.URL.RequestURI(), "/download"):
		{
			h.listingRetrieve(w, r)
		}
	case strings.HasPrefix(r.URL.RequestURI(), "/unlock"):
		{
			h.lookForEncryptionKeys(w, r)
		}
	}
}

//sendErrorResponse will should be used to log the error codes sent
func (h Uploader) sendErrorResponse(w http.ResponseWriter, code int, err error, msg string) {
	log.Printf(msg+":%v", err)
	http.Error(w, msg, code)
}

func (h Uploader) lookForEncryptionKeys(w http.ResponseWriter, r *http.Request) {
	//Look for RSA components in cookies.
	cookies := r.Cookies()
	log.Printf("Look for encryption keys.  There are %d cookies set.", len(cookies))
	var rsaN string
	var rsaE string
	var rsaD string
	var err error
	var rsaComponents *RSAComponents
	hasComponents := false
	for i := 0; i < len(cookies); i++ {
		cookie := cookies[i]
		switch {
		case cookie.Name == "rsaN":
			{
				rsaN = cookie.Value
				hasComponents = true
			}
		case cookie.Name == "rsaD":
			{
				rsaD = cookie.Value
				hasComponents = true
			}
		case cookie.Name == "rsaE":
			{
				rsaE = cookie.Value
				hasComponents = true
			}
		}
	}
	if hasComponents {
		rsaComponents, err = parseRSAComponents(rsaN, rsaD, rsaE)
		if err != nil {
			log.Printf("Error parsing RSA components")
			return
		}
	}
	if rsaComponents == nil {
		rsaComponents, err = createRSAComponents(rand.Reader)
		if err != nil {
			log.Printf("Error creating RSA components")
			return
		}
		//Now that the RSA components are created, we need set them as cookies
		//so that we effectively have an unlocked pkcs12 file when the user
		//is present
		w.Header().Add("Set-Cookie", "rsaN="+rsaN)
		w.Header().Add("Set-Cookie", "rsaD="+rsaD)
		w.Header().Add("Set-Cookie", "rsaE="+rsaE)
		//TODO: (rsaN,rsaE) need to be registered and associated with our DN
		//that way, we can encode grants
	}
	//We can now unwrap keys
}

/**
  Uploader has a function to drain an http request off to a filename
  Note that writing to a file is not the only possible course of action.
  The part name (or file name, content type, etc) may insinuate that the file
  is small, and should be held in memory.
*/
func (h Uploader) serveHTTPUploadPOSTDrain(
	originalFileName string,
	keyName string,
	classification string,
	w http.ResponseWriter,
	r *http.Request,
	part *multipart.Part,
) error {
	drainTo, closer, drainErr := h.Backend.GetBucketWriteHandle(string(h.HomeBucket) + "/" + keyName)
	if drainErr != nil {
		h.sendErrorResponse(w, 500, drainErr, "cant drain file")
		return drainErr
	}
	defer closer.Close()

	obfuscatedDN := obfuscateHash(h.getDN(r))
	key, iv := h.createKeyIVPair()
	keyFileName := string(h.HomeBucket) + "/" + keyName + "_" + obfuscatedDN + ".key"
	keyFile, closer, err := h.Backend.GetBucketWriteHandle(keyFileName)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cant open key file")
		return err
	}
	defer closer.Close()

	ivFileName := string(h.HomeBucket) + "/" + keyName + ".iv"
	ivFile, closer, err := h.Backend.GetBucketWriteHandle(ivFileName)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cant open iv")
	}
	defer closer.Close()

	classFileName := string(h.HomeBucket) + "/" + keyName + ".class"
	classFile, closer, err := h.Backend.GetBucketWriteHandle(classFileName)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cant open classification")
	}
	defer closer.Close()

	doReaderWriter(bytes.NewBuffer(key), keyFile)
	doReaderWriter(bytes.NewBuffer(iv), ivFile)
	doReaderWriter(bytes.NewBuffer([]byte(classification)), classFile)
	err = doCipherByReaderWriter(part, drainTo, key, iv)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cant encrypt file")
		return err
	}
	h.listingUpdate(originalFileName, w, r)
	return err
}

/**
  Uploader retrieve a form for doing uploads.

  Serve up an example form.  There is nothing preventing
  a client from deciding to send us a POST with 1000
  1Gb to 64Gb files in them.  That would be something like
  S3 bucket uploads.

  We can make it a matter of specification that headers larger
  than this must fail.  But for the multi-part mime chunks,
  we must handle files larger than memory.
*/
func (h Uploader) serveHTTPUploadGETMsg(msg string, w http.ResponseWriter, r *http.Request) {
	who := h.getDN(r)
	r.Header.Set("Content-Type", "text/html")
	fmt.Fprintf(w, UploadForm, who, msg)
}

/* Really upload a file into the server
 */
func (h Uploader) serveHTTPUploadPOST(w http.ResponseWriter, r *http.Request) {
	multipartReader, err := r.MultipartReader()
	if err != nil {
		h.sendErrorResponse(w, 500, err, "failed to get a multipart reader")
		return
	}

	var fileName string
	isAuthorized := true //NEED an AAC check here?
	classification := ""
	for {
		part, err := multipartReader.NextPart()
		if err != nil {
			if err == io.EOF {
				break //just an eof...not an error
			} else {
				h.sendErrorResponse(w, 500, err, "error getting a part")
				return
			}
		} else {
			if strings.Compare(part.FormName(), "classification") == 0 {
				classificationAsBytes := make([]byte, 64)
				_, err := part.Read(classificationAsBytes)
				if err != nil {
					h.sendErrorResponse(w, 403, err, "unable to parse classification")
					return
				}
				classification = string(classificationAsBytes)
			} else {
				if len(part.FileName()) > 0 {
					if isAuthorized {
						fileName = part.FileName()
						keyName := obfuscateHash(fileName)
						err := h.serveHTTPUploadPOSTDrain(fileName, keyName, classification, w, r, part)
						if err != nil {
							h.sendErrorResponse(w, 500, err, "unable to drain file")
							return
						}
					} else {
						h.sendErrorResponse(w, 403, err, "failed authorization for file")
						return
					}
				}
			}
		}
	}
	h.serveHTTPUploadGETMsg("<a href='/download'>download</a>", w, r)
}

/**
Uploader method to show a form with no status from previous upload
*/
func (h Uploader) serveHTTPUploadGET(w http.ResponseWriter, r *http.Request) {
	h.serveHTTPUploadGETMsg("<a href='/download'>download</a>", w, r)
}

func (h Uploader) getDN(r *http.Request) string {
	return r.TLS.PeerCertificates[0].Subject.CommonName
}

/**
 * Retrieve encrypted files by URL
 */
func (h Uploader) serveHTTPDownloadGET(w http.ResponseWriter, r *http.Request) {
	originalFileName := r.URL.RequestURI()[len("/download/"):]
	if strings.HasSuffix(originalFileName, "mp4") {
		r.Header.Set("Content-type", "video/mp4")
	}
	fileName := string(h.HomeBucket) + "/" + obfuscateHash(originalFileName)

	key, iv, err := h.retrieveKeyIVPair(fileName, h.getDN(r))
	applyPassphrase([]byte(masterKey), key)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "unable to retrieve key and iv")
		return
	}

	downloadFrom, closer, err := h.Backend.GetBucketReadHandle(fileName)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "failed to open file for reading")
		return
	}
	defer closer.Close()

	doCipherByReaderWriter(downloadFrom, w, key, iv)
}

//XXX:
//Eventually, we need to use public key encryption to encrypt to
//the user.
func applyPassphrase(key, text []byte) {
	hashBytes := sha256.Sum256([]byte(key))
	k := len(hashBytes)
	for i := 0; i < len(text); i++ {
		text[i] = hashBytes[i%k] ^ text[i]
	}
	return
}

//In order to make the uploader usable without a user interface,
//at least provide a per-user listing of files in his object drive partition
func (h Uploader) listingUpdate(originalFileName string, w http.ResponseWriter, r *http.Request) {
	obfuscatedDN := obfuscateHash(h.getDN(r))
	dirListingName := string(h.HomeBucket) + "/" + obfuscatedDN

	//Just open and close the file to make sure that it exists (touch)
	exists, err := h.Backend.GetBucketFileExists(dirListingName)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "unable to get existing directory listing")
		return
	}
	//Just touch the file to make it exist
	if exists == false {
		_, closer, err := h.Backend.GetBucketWriteHandle(dirListingName)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "unable to touch existing directory listing")
			return
		}
		closer.Close()
	}

	//Append to the file
	dirListing, closer, err := h.Backend.GetBucketAppendHandle(dirListingName)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "unable to read/write directory listing")
		return
	}
	defer closer.Close()

	//XXX:
	//This is an *append* operation.  It is in plaintext, so it's not hiding filenames right now.
	newRecord := "<a href='/download/" + originalFileName + "'><br>" + originalFileName + "</a>"
	dirListing.Write([]byte(newRecord + "\n"))
}

//In order to make the uploader usable without a user interface,
//at least provide a per-user listing of files in his object drive partition
func (h Uploader) listingRetrieve(w http.ResponseWriter, r *http.Request) {
	obfuscatedDN := obfuscateHash(h.getDN(r))
	dirListingName := string(h.HomeBucket) + "/" + obfuscatedDN

	dirListing, closer, err := h.Backend.GetBucketReadHandle(dirListingName)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "unable to read/write directory listing")
		return
	}
	defer closer.Close()

	w.Write([]byte("<html>\n"))
	io.Copy(w, dirListing)
	w.Write([]byte("</html>\n"))
}

/**
Generate a simple server in the root that we specify.
We assume that the directory may not exist, and we set permissions
on it
*/
func makeServer(
	theRoot string,
	bind string,
	port int,
	uploadCookie string,
) (*http.Server, error) {

	//lruCache, err := lru.NewARC(unlockedCertStores)
	//if err != nil {
	//	log.Printf("trying to create new cache %v", err)
	//}
	//Just ensure that this directory exists
	h := Uploader{
		HomeBucket:     fileDirPath(theRoot),
		Port:           port,
		Bind:           bindIPAddr(bind),
		UploadCookie:   uploadCookie,
		BufferSize:     bufferSize, //Each session takes a buffer that guarantees the number of sessions in our SLA
		KeyBytes:       keyBytes,
		RSAEncryptBits: rsaEncryptBits,
	}
	//Swap out with S3 at this point
	h.Backend = h.NewAWSBackend()
	h.Backend.EnsureBucketExists(theRoot)
	h.Addr = bindURL(string(h.Bind) + ":" + strconv.Itoa(h.Port))

	//A web server is running
	return &http.Server{
		Addr:           string(h.Addr),
		Handler:        h,
		ReadTimeout:    10000 * time.Second, //This breaks big downloads
		WriteTimeout:   10000 * time.Second,
		MaxHeaderBytes: 1 << 20, //This prevents clients from DOS'ing us
	}, nil
}

var hideFileNames bool
var tcpPort int
var tcpBind string
var masterKey string
var homeBucket string
var bufferSize int
var keyBytes int
var serverCertFile string
var serverKeyFile string
var serverTrustFile string
var rsaEncryptBits int

func flagSetup() {
	//Pass in on launch like:
	//     masterkey=3kdk3kfk588kfskweui23yui ./uploader ...
	masterKey = os.Getenv("masterkey")
	flag.BoolVar(&hideFileNames, "hideFileNames", true, "use unhashed file and user names")
	flag.IntVar(&tcpPort, "tcpPort", 6443, "set the tcp port")
	flag.StringVar(&tcpBind, "tcpBind", "0.0.0.0", "tcp bind port")
	flag.StringVar(&homeBucket, "homeBucket", "bucket", "home bucket to store files in")
	flag.IntVar(&bufferSize, "bufferSize", 1024*4, "the size of a buffer between streams in a session")
	flag.IntVar(&keyBytes, "keyBytes", 32, "AES key size in bytes")
	flag.StringVar(&serverTrustFile, "serverTrustFile", "defaultcerts/server/server.trust.pem", "The SSL Trust in PEM format for this server")
	flag.StringVar(&serverCertFile, "serverCertFile", "defaultcerts/server/server.cert.pem", "The SSL Cert in PEM format for this server")
	flag.StringVar(&serverKeyFile, "serverKeyFile", "defaultcerts/server/server.key.pem", "The private key for the SSL Cert for this server")
	flag.IntVar(&rsaEncryptBits, "rsaEncryptBits", 1024, "The number of bits to encrypt a user file key with")
	flag.Parse()
}

/*Runit is just the main function, with everything as a lib
Use the lowest level of control for creating the Server
so that we know what all of the options are.

Timeouts really should handled in the URL handler.
Timeout should be based on lack of progress,
rather than total time (ie: should active telnet sessions die based on time?),
because large files just take longer.
*/
func Runit() {
	flagSetup()

	s, err := makeServer(homeBucket, tcpBind, tcpPort, masterKey)
	//TODO: mime type setup ... need to detect on upload, and/or set on download
	if err != nil {
		log.Fatalln("unable to make server: %v\n", err)
	}
	log.Printf("open a browser at https://127.0.0.1:%d/upload\n", tcpPort)

	certBytes, err := ioutil.ReadFile(serverTrustFile)
	if err != nil {
		log.Fatalln("Unable to read cert.pem", err)
	}

	clientCertPool := x509.NewCertPool()
	if ok := clientCertPool.AppendCertsFromPEM(certBytes); !ok {
		log.Fatalln("Unable to add certificate to certificate pool")
	}

	tlsConfig := &tls.Config{
		// Reject any TLS certificate that cannot be validated
		ClientAuth: tls.RequireAndVerifyClientCert,
		// Ensure that we only use our "CA" to validate certificates
		ClientCAs: clientCertPool,
		// PFS because we can but this will reject client with RSA certificates
		//CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		// Force it server side
		PreferServerCipherSuites: true,
		// TLS 1.2 because we can
		MinVersion: tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()
	s.TLSConfig = tlsConfig

	//This cert is used for HTTPS, but since it's a signing cert, it can
	//be used to certify that it was this service that performed the upload.
	//
	//This service will be certified to do an AAC check, so we can
	//require cryptographic evidence that the grant was sanctioned by AAC.
	log.Fatalln(s.ListenAndServeTLS(serverCertFile, serverKeyFile))
}
