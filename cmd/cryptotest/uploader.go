package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/golang-lru"
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

/*
 * These are the templates to give a basic user interface.
 */

var indexForm = `
<html>
	<head><title>OD Uploader</title>
	<body>
		<a href='/upload'>Upload</a>
	</body>
</html>
`

var uploadForm = `
<html>
  <head><title>Upload A File</title>
	<body>
		%s
		%s
		<br>
		<form action='/upload' method='POST' enctype='multipart/form-data'>
			<select name='classification'>
				<option value='U'>Unclassified</option>
				<option value='C'>Classified</option>
				<option value='S'>Secret</option>
				<option value='T'>Top Secret</option>
			</select>
			The File:<input name='theFile' type='file'>
			<input type='submit'>
		</form>
	</body>
</html>
`

/* ServeHTTP handles the routing of requests
 */
func (h uploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.RequestURI() == "/upload":
		{
			switch {
			case r.Method == "GET":
				h.serveHTTPUploadGET(w, r)
			case r.Method == "POST":
				h.serveHTTPUploadPOST(w, r)
			}
		}
	case strings.HasPrefix(r.URL.RequestURI(), "/download/"):
		{
			h.serveHTTPDownloadGET(w, r)
		}
	case strings.HasPrefix(r.URL.RequestURI(), "/download"):
		{
			h.listingRetrieve(w, r)
		}
	}
}

type fileDirPath string
type bindIPAddr string
type bindURL string

/**
  Uploader is a special type of Http server.
  Put any config state in here.
  The point of this server is to show how
  upload and download can be extremely efficient
  for large files.
*/
type uploader struct {
	HomeBucket         fileDirPath
	Port               int
	Bind               bindIPAddr
	Addr               bindURL
	UploadCookie       string
	BufferSize         int
	KeyBytes           int
	UnlockedCertStores *lru.ARCCache
	RSAEncryptBits     int
}

//Generate unique opaque names for uploaded files
func obfuscateHash(key string) string {
	if hideFileNames {
		hashBytes := sha256.Sum256([]byte(key))
		keyString := base64.StdEncoding.EncodeToString(hashBytes[:])
		return strings.Replace(strings.Replace(keyString, "/", "~", -1), "=", "Z", -1)
	}
	return key
}

// CountingStreamReader takes statistics as it writes
type CountingStreamReader struct {
	S cipher.Stream
	R io.Reader
}

// Read takes statistics as it writes
func (r CountingStreamReader) Read(dst []byte) (n int, err error) {
	n, err = r.R.Read(dst)
	r.S.XORKeyStream(dst[:n], dst[:n])
	return
}

// CountingStreamWriter keeps statistics as it writes
type CountingStreamWriter struct {
	S     cipher.Stream
	W     io.Writer
	Error error
}

func (w CountingStreamWriter) Write(src []byte) (n int, err error) {
	c := make([]byte, len(src))
	w.S.XORKeyStream(c, src)
	n, err = w.W.Write(c)
	if n != len(src) {
		if err == nil {
			err = io.ErrShortWrite
		}
	}
	return
}

// Close closes underlying stream
func (w CountingStreamWriter) Close() error {
	if c, ok := w.W.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (h uploader) createKeyIVPair() (key []byte, iv []byte) {
	key = make([]byte, h.KeyBytes)
	rand.Read(key)
	iv = make([]byte, aes.BlockSize)
	rand.Read(iv)
	return
}

func (h uploader) retrieveKeyIVPair(fileName string, r *http.Request) (key []byte, iv []byte, ret error) {
	keyFileName := fileName + "_" + obfuscateHash(h.getDN(r)) + ".key"
	ivFileName := fileName + ".iv"

	log.Printf("opening: %s\n", keyFileName)
	keyFile, err := os.Open(keyFileName)
	if err != nil {
		return key, iv, err
	}
	defer keyFile.Close()
	key = make([]byte, h.KeyBytes)
	keyFile.Read(key)

	log.Printf("opening: %s\n", ivFileName)
	ivFile, err := os.Open(ivFileName)
	if err != nil {
		return key, iv, err
	}
	defer ivFile.Close()
	iv = make([]byte, aes.BlockSize)
	ivFile.Read(iv)
	scramble([]byte(masterKey), key)
	return key, iv, ret
}

func doCipherByReaderWriter(inFile io.Reader, outFile io.Writer, key []byte, iv []byte) error {
	log.Printf("creating cipher with key of length %d\n", len(key))
	writeCipher, err := aes.NewCipher(key)
	log.Print("got it")
	if err != nil {
		log.Printf("%v", err)
		return err
	}
	log.Printf("creating block mode with iv of length %d\n", len(iv))
	writeCipherStream := cipher.NewCTR(writeCipher, iv[:])
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	log.Print("copying stream cipher")
	reader := &CountingStreamReader{S: writeCipherStream, R: inFile}
	_, err = io.Copy(outFile, reader)
	if err != nil {
		log.Printf("%v", err)
	}
	return err
}

func doReaderWriter(inFile io.Reader, outFile io.Writer) error {
	_, err := io.Copy(outFile, inFile)
	return err
}

func (h uploader) getDN(r *http.Request) string {
	return r.TLS.PeerCertificates[0].Subject.CommonName
}

/**
  Uploader has a function to drain an http request off to a filename
  Note that writing to a file is not the only possible course of action.
  The part name (or file name, content type, etc) may insinuate that the file
  is small, and should be held in memory.
*/
func (h uploader) serveHTTPUploadPOSTDrain(
	originalFileName string,
	keyName string,
	classification string,
	w http.ResponseWriter,
	r *http.Request,
	part *multipart.Part,
) error {
	log.Printf("read part %s", keyName)
	drainTo, drainErr := os.Create(string(h.HomeBucket) + "/" + keyName)
	if drainErr != nil {
		log.Printf("error draining file: %v", drainErr)
	}
	defer drainTo.Close()
	obfuscatedDN := obfuscateHash(h.getDN(r))
	key, iv := h.createKeyIVPair()
	keyFileName := string(h.HomeBucket) + "/" + keyName + "_" + obfuscatedDN + ".key"
	keyFile, err := os.Create(keyFileName)
	defer keyFile.Close()
	if err != nil {
		log.Printf("Could not open key file")
		return err
	}
	ivFileName := string(h.HomeBucket) + "/" + keyName + ".iv"
	ivFile, err := os.Create(ivFileName)
	defer ivFile.Close()
	classFileName := string(h.HomeBucket) + "/" + keyName + ".class"
	classFile, err := os.Create(classFileName)
	defer classFile.Close()
	doReaderWriter(bytes.NewBuffer(key), keyFile)
	doReaderWriter(bytes.NewBuffer(iv), ivFile)
	doReaderWriter(bytes.NewBuffer([]byte(classification)), classFile)
	err = doCipherByReaderWriter(part, drainTo, key, iv)
	if err != nil {
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
func (h uploader) serveHTTPUploadGETMsg(msg string, w http.ResponseWriter, r *http.Request) {
	log.Print("get an upload get")
	who := h.getDN(r)
	r.Header.Set("Content-Type", "text/html")
	fmt.Fprintf(w, uploadForm, who, msg)
}

func (h uploader) serveHTTPUploadPOST(w http.ResponseWriter, r *http.Request) {
	multipartReader, err := r.MultipartReader()
	if err != nil {
		log.Printf("failed to get a multipart reader %v", err)
		http.Error(w, "failed to get a multipart reader", 500)
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
				log.Printf("error getting a part %v", err)
				http.Error(w, "error getting a part", 500)
				return
			}
		} else {
			if strings.Compare(part.FormName(), "classification") == 0 {
				classificationAsBytes := make([]byte, 64)
				_, err := part.Read(classificationAsBytes)
				if err != nil {
					log.Printf("Unable to parse classification: %v", err)
					http.Error(w, "Unable to parse classification", 500)
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
							log.Printf("error draining part: %v", err)
						}
					} else {
						log.Printf("failed authorization for file")
						http.Error(w, "failed authorization for file", 400)
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
func (h uploader) serveHTTPUploadGET(w http.ResponseWriter, r *http.Request) {
	h.serveHTTPUploadGETMsg("", w, r)
}

func (h uploader) serveHTTPDownloadGET(w http.ResponseWriter, r *http.Request) {
	//"mp4 -> video/mp4"
	originalFileName := r.URL.RequestURI()[len("/download/"):]
	fileName := string(h.HomeBucket) + "/" + obfuscateHash(originalFileName)
	log.Printf("download request for %s", originalFileName)
	key, iv, err := h.retrieveKeyIVPair(fileName, r)
	scramble([]byte(masterKey), key)
	if err != nil {
		log.Print("Unable to retrieve iv and key")
	}
	downloadFrom, err := os.Open(fileName)
	if err != nil {
		log.Print("failed to open file for reading")
		http.Error(w, "failed to open file for reading", 500)
		return
	}
	defer downloadFrom.Close()
	doCipherByReaderWriter(downloadFrom, w, key, iv)
}

//A retarded xor key scramble for now - at least we are xor'ing with
//a random key
func scramble(key, text []byte) {
	k := len(key)
	for i := 0; i < len(text); i++ {
		text[i] = key[i%k] ^ text[i]
	}
	return
}

//In order to make the uploader usable without a user interface,
//at least provide a per-user listing of files in his object drive partition
func (h uploader) listingUpdate(originalFileName string, w http.ResponseWriter, r *http.Request) {
	log.Printf("insert " + originalFileName + " into listing")
	obfuscatedDN := obfuscateHash(h.getDN(r))
	dirListingName := string(h.HomeBucket) + "/" + obfuscatedDN
	var dirListing *os.File
	var err error
	if _, err = os.Stat(dirListingName); os.IsNotExist(err) {
		dirListing, err = os.Create(dirListingName)
	} else {
		dirListing, err = os.OpenFile(dirListingName, os.O_RDWR|os.O_APPEND, 0600)
	}
	if err != nil {
		log.Printf("unable to read/write directory listing %v", err)
	}
	defer dirListing.Close()
	newRecord := "<a href='/download/" + originalFileName + "'><hr>" + originalFileName + "</a>"
	dirListing.Write([]byte(newRecord + "\n"))
}

//In order to make the uploader usable without a user interface,
//at least provide a per-user listing of files in his object drive partition
func (h uploader) listingRetrieve(w http.ResponseWriter, r *http.Request) {
	log.Printf("getting a listing")
	obfuscatedDN := obfuscateHash(h.getDN(r))
	dirListingName := string(h.HomeBucket) + "/" + obfuscatedDN
	var dirListing *os.File
	var err error
	dirListing, err = os.Open(dirListingName)
	if err != nil {
		log.Printf("unable to read directory listing %v", err)
	}
	defer dirListing.Close()
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

	lruCache, err := lru.NewARC(unlockedCertStores)
	if err != nil {
		log.Printf("trying to create new cache %v", err)
	}
	//Just ensure that this directory exists
	os.Mkdir(theRoot, 0700)
	h := uploader{
		HomeBucket:         fileDirPath(theRoot),
		Port:               port,
		Bind:               bindIPAddr(bind),
		UploadCookie:       uploadCookie,
		BufferSize:         bufferSize, //Each session takes a buffer that guarantees the number of sessions in our SLA
		KeyBytes:           keyBytes,
		UnlockedCertStores: lruCache,
		RSAEncryptBits:     rsaEncryptBits,
	}
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

func generateSession(account string) *s3.S3 {
	sessionConfig := &aws.Config{
		Credentials: credentials.NewSharedCredentials("", account),
	}
	sess := session.New(sessionConfig)
	svc := s3.New(sess)
	return svc
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
var unlockedCertStores int
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
	flag.IntVar(&unlockedCertStores, "unlockedCertStores", 10000, "The number of unlocked cert stores we allow in the system")
	flag.IntVar(&rsaEncryptBits, "rsaEncryptBits", 1024, "The number of bits to encrypt a user file key with")
	flag.Parse()
}

/**
  Use the lowest level of control for creating the Server
  so that we know what all of the options are.

  Timeouts really should handled in the URL handler.
  Timeout should be based on lack of progress,
  rather than total time (ie: should active telnet sessions die based on time?),
  because large files just take longer.
*/
func main() {
	flagSetup()

	s, err := makeServer(homeBucket, tcpBind, tcpPort, masterKey)
	//TODO: mime type setup ... need to detect on upload, and/or set on download
	if err != nil {
		log.Printf("unable to make server: %v\n", err)
		return
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
	log.Fatal(s.ListenAndServeTLS(serverCertFile, serverKeyFile))
}
