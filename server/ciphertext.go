package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"

	"decipher.com/object-drive-server/config"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"
)

var (
	// PeerMap is repopulated by a callback that knows when the odrive membership group changes
	PeerMap = make(map[string]*PeerMapData)
)

// PeerMapData is the information we need to create a connection to a peer
// to get ciphertext
type PeerMapData struct {
	Host    string
	Port    int
	CA      string
	Cert    string
	CertKey string
	Client  *http.Client
}

//
// If a peer can't get ciphertext from PermanentStorage, then it can ask around to see who has it.
// If we get asked, we can serve it back to the caller.  If we don't ask peers, we can be
// stuck trying to get the ciphertext from PermanentStorage in a very long stall that will time out.
//
// Also if PermanentStorage is disabled with a load balanced setup, the ciphertext would not come back at all
// without p2p requesting.
//
func (h AppServer) getCiphertext(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	if r.Header.Get("USER_DN") != "P2P" {
		return NewAppError(403, fmt.Errorf("p2p required to get ciphertext"), "forbidden")
	}
	//We are getting a p2p ciphertext request, so that we can handle getting range requests
	//before a file can make it into PermanentStorage
	logger := LoggerFromContext(ctx)
	//Ask a drain provider directly to give us a particular ciphertext.
	captureGroups, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return NewAppError(400, nil, "unparseable uri parameters")
	}

	//Specify which ciphertext out of which drain provider we are looking for
	CiphertextCacheSelector := captureGroups["selector"]
	rName := FileId(captureGroups["rname"])
	dp := FindCiphertextCache(CiphertextCacheSelector)

	//If there is a byte range, then use it.
	startAt := int64(0)
	byteRange, err := extractByteRange(r)
	if err != nil {
		return NewAppError(400, err, "byte range parse fail")
	}
	//We just want to know where to start from, and stream the whole file
	//until the client stops reading it.
	if byteRange != nil {
		startAt = byteRange.Start
	}

	//Send back the byte range asked for
	f, length, err := useLocalFile(logger, dp, rName, startAt)
	if err != nil {
		//Keep it quiet in the case of not found
		return NewAppError(500, err, "error looking in p2p cache")
	}
	if f == nil {
		return NewAppError(204, nil, "not in this p2p cache")
	}
	if length < 0 {
		logger.Error("p2p bad length", zap.Int64("Content-Length", length))
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	//w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
	byteCount, err := io.Copy(w, f)

	//It is perfectly normal for a client to only pull part of the data and cut us off
	if err != nil && strings.Contains(err.Error(), "write: connection reset by peer") == false {
		logger.Info("p2p copy failure", zap.String("err", err.Error()), zap.Int64("bytes", byteCount))
	}
	return nil
}

func newTLSConfig(trustPath, certPath, keyPath string) (*tls.Config, error) {
	trustBytes, err := ioutil.ReadFile(trustPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing CA trust %s: %v", trustPath, err)
	}
	trustCertPool := x509.NewCertPool()
	if !trustCertPool.AppendCertsFromPEM(trustBytes) {
		return nil, fmt.Errorf("Error adding CA trust to pool: %v", err)
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing cert: %v", err)
	}
	cfg := tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                trustCertPool,
		InsecureSkipVerify:       true,
		ServerName:               "twl-server-generic2",
		PreferServerCipherSuites: true,
	}
	cfg.BuildNameToCertificate()

	return &cfg, nil
}

// useP2PFile is similar to useLocalFile, except it searches peer caches for our ciphertext.
// It is better to do this than it is to stall while the file moves to S3.
//
// It is the CALLER's responsibility to close io.ReadCloser !!
func useP2PFile(logger zap.Logger, CiphertextCacheSelector string, rName FileId, begin int64) (io.ReadCloser, error) {
	cfgPort, _ := strconv.Atoi(config.Port)
	for _, peer := range PeerMap {
		//If this is NOT our own entry
		if peer != nil && (peer.Host != config.MyIP || peer.Port != cfgPort) {
			//Ensure that we have a connection to the peer
			url := fmt.Sprintf("https://%s:%d/ciphertext/%s/%s", peer.Host, peer.Port, CiphertextCacheSelector, string(rName))

			//Set up a transport to connect to peer if there isn't one
			var conf *tls.Config
			var err error
			if peer.Client == nil {
				conf, err = newTLSConfig(peer.CA, peer.Cert, peer.CertKey)
				if err != nil {
					logger.Warn("p2p cannot connect", zap.String("url", url), zap.String("err", err.Error()))
				}
				peer.Client = &http.Client{
					Transport: &http.Transport{
						DialTLS: func(network, address string) (net.Conn, error) {
							return tls.Dial("tcp", fmt.Sprintf("%s:%d", peer.Host, peer.Port), conf)
						},
					},
				}
			}
			if peer.Client != nil {
				req, err := http.NewRequest("GET", url, nil)
				if err == nil {
					rangeResponse := fmt.Sprintf("bytes=%d-", begin)
					req.Header.Set("Range", rangeResponse)
					//P2P does not pass through nginx, so only this value can happen P2P, and we use
					//2 way SSL to enforce only peers connecting to us.
					req.Header.Set("USER_DN", "P2P")
					res, err := peer.Client.Do(req)
					if err == nil && res != nil && res.StatusCode == http.StatusOK {
						return res.Body, nil
					}
				}
				if err != nil {
					logger.Error("p2p cannot connect to peer", zap.String("err", err.Error()))
				}
			}
		}
	}
	return nil, nil
}
