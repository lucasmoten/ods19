package ciphertext

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/config"

	"sync"
)

var (
	// peerMap is repopulated by a callback that knows when the odrive membership group changes
	peerMap            = make(map[string]*PeerMapData) //Atomically updated
	connectionMap      = make(map[string]*http.Client) //Locked for add and remove - no IO under these locks!
	connectionMapMutex = &sync.RWMutex{}
)

// PeerMapData is the information we need to create a connection to a peer
// to get ciphertext
type PeerMapData struct {
	Host    string
	Port    int
	CA      string
	Cert    string
	CertKey string
}

// CreateRandomName gives each file a random name
func CreateRandomName() string {
	// NOTE: This length affects the rname a.k.a. contentConnector length when in hexadecimal format.  If this is altered
	// the API request for the Ciphertext route regular expression may need updated.
	key := make([]byte, 26)
	rand.Read(key)
	return hex.EncodeToString(key)
}

// ScheduleSetPeers sets a new peer set - there is only one thread that calls this
func ScheduleSetPeers(newPeerMap map[string]*PeerMapData) {
	setPeers(newPeerMap, peerMap)
}

// setPeers calculates which connections can be deleted and sets the new peermap
func setPeers(newPeerMap map[string]*PeerMapData, oldPeerMap map[string]*PeerMapData) {

	//Delete old items from the connection map - this just needs to be done eventually
	connectionMapMutex.Lock()

	//Compute deleted items by the diff
	var deletedPeerKeys []string
	for oldPeerKey := range oldPeerMap {
		peer := newPeerMap[oldPeerKey]
		if peer == nil {
			deletedPeerKeys = append(deletedPeerKeys, oldPeerKey)
		}
	}

	//These are never mutated, so no problem
	peerMap = newPeerMap
	for _, k := range deletedPeerKeys {
		delete(connectionMap, k)
	}
	connectionMapMutex.Unlock()
}

// NewTLSClientConn is a wrapper preparing a http.Client connection setup to use the provided PKI credentials, and
// connect to a specific server host and port.
func NewTLSClientConn(trustPath, certPath, keyPath, serverName, host, port string, insecure bool) (*http.Client, error) {
	conf, err := config.NewTLSClientConfig(trustPath, certPath, keyPath, serverName, insecure)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{
			DialTLS: func(network, address string) (net.Conn, error) {
				return tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), conf)
			},
		},
	}, nil
}

// UseLocalFile returns a handle to either the FileStateCached file or FileStateUploaded file
//  It is the caller's responsibility to close the file handle
func UseLocalFile(logger *zap.Logger, d CiphertextCache, rName FileId, cipherStartAt int64) (*os.File, int64, error) {
	var cipherFile *os.File
	var err error
	var length int64
	tm := time.Now().UTC()
	cipherFilePathUploaded := d.Resolve(NewFileName(rName, FileStateUploaded))
	cipherFilePathCached := d.Resolve(NewFileName(rName, FileStateCached))

	//Try the uploaded file
	logger.Debug("useLocalFile checking for FileStateUploaded")
	info, ierr := d.Files().Stat(cipherFilePathUploaded)
	if ierr == nil {
		length = info.Size() - cipherStartAt
	}
	cipherFile, err = d.Files().Open(cipherFilePathUploaded)
	if err != nil {
		// DIMEODS-1262 - Ensure file closed if not nil
		if cipherFile != nil {
			cipherFile.Close()
		}
		//Try the cached file
		logger.Debug("useLocalFile checking for FileStateCached")
		info, ierr := d.Files().Stat(cipherFilePathCached)
		if ierr == nil {
			length = info.Size() - cipherStartAt
		}
		cipherFile, err = d.Files().Open(cipherFilePathCached)
		if err != nil {
			// DIMEODS-1262 - Ensure file closed if not nil
			if cipherFile != nil {
				cipherFile.Close()
			}
			if os.IsNotExist(err) {
				return nil, -1, nil
			}
			return nil, -1, err
		}
	} else {
		// Update timestamp on file that is being used
		d.Files().Chtimes(cipherFilePathUploaded, tm, tm)
	}
	//We have a file handle.  Seek to where we should start reading the cipher.
	logger.Debug("useLocalFile found a filehandle")
	_, err = cipherFile.Seek(cipherStartAt, 0)
	if err != nil {
		logger.Error("useLocalFile failed to seek", zap.Int64("cipherStartAt", cipherStartAt))
		cipherFile.Close()
		return nil, -1, err
	}
	//Update the timestamps to note the last time it was used
	// This is done here, as well as successful end just in case of failures midstream.
	logger.Debug("useLocalFile is touching file timestamp", zap.String("cachedfile", string(cipherFilePathCached)))
	d.Files().Chtimes(cipherFilePathCached, tm, tm)

	return cipherFile, length, nil
}

// useP2PFile is similar to useLocalFile, except it searches peer caches for our ciphertext.
// It is better to do this than it is to stall while the file moves to S3.
//
// It is the CALLER's responsibility to close io.ReadCloser !!
func useP2PFile(logger *zap.Logger, zone CiphertextCacheZone, rName FileId, begin int64) (io.ReadCloser, error) {
	if strings.ToLower(os.Getenv(config.OD_PEER_ENABLED)) != "true" {
		return nil, nil
	}
	//Iterate over the current value of peerMap.  Do NOT lock this loop, as there is long IO in here.
	connectionMapMutex.RLock()
	thisMap := peerMap
	connectionMapMutex.RUnlock()
	for peerKey, peer := range thisMap {
		//If this is NOT our own entry
		if peer != nil && (peer.Host != config.MyIP) {
			//Ensure that we have a connection to the peer
			url := fmt.Sprintf("https://%s:%d/ciphertext/%s/%s", peer.Host, peer.Port, string(zone), string(rName))

			//Set up a transport to connect to peer if there isn't one
			var err error
			var conn *http.Client

			//Get the existing connection - this is the one we are hitting all the time, so it's important that it's a read-lock
			//(because we are almost never writing, except for a brief flash when zk nodes change, which is exceedingly rare)
			connectionMapMutex.RLock()
			conn = connectionMap[peerKey]
			connectionMapMutex.RUnlock()

			if conn == nil {
				peerCN := config.GetEnvOrDefault(config.OD_PEER_CN, "")
				insecureSkipVerify := config.GetEnvOrDefault(config.OD_PEER_INSECURE_SKIP_VERIFY, "false") == "true"
				conn, err = NewTLSClientConn(
					peer.CA,
					peer.Cert,
					peer.CertKey,
					peerCN,
					peer.Host,
					fmt.Sprintf("%d", peer.Port),
					insecureSkipVerify,
				)
				if err != nil {
					logger.Warn("p2p cannot connect", zap.String("url", url), zap.Error(err))
				}

				//Set the new connection if we got one
				connectionMapMutex.Lock()
				connectionMap[peerKey] = conn
				connectionMapMutex.Unlock()
			}
			if conn != nil {
				req, err := http.NewRequest("GET", url, nil)
				if err == nil {
					rangeResponse := fmt.Sprintf("bytes=%d-", begin)
					req.Header.Set("Range", rangeResponse)
					// P2P is direct and does not route through edge/gateway, so only this value can happen P2P.
					// We use 2 way SSL to enforce only peers connecting to us.
					PeerSignifier := config.GetEnvOrDefault(config.OD_PEER_SIGNIFIER, "P2P")
					req.Header.Set("USER_DN", PeerSignifier)
					res, err := conn.Do(req) //connectionMap[peerKey].Do(req)
					if err != nil {
						// DIMEODS-1262 - read and close response that we aren't going to use to avoid file leak
						if res != nil && res.Body != nil {
							ioutil.ReadAll(res.Body)
							res.Body.Close()
						}
						logger.Error("p2p cannot connect to peer (failed do)",
							zap.Error(err),
							zap.String("peerKey", peerKey),
							zap.String("peerCN", config.GetEnvOrDefault(config.OD_PEER_CN, "")),
							zap.String("peer.CA", peer.CA),
							zap.String("peer.Cert", peer.Cert),
						)
						// and set connection as nil to force it to be re-established
						connectionMapMutex.Lock()
						connectionMap[peerKey] = nil
						connectionMapMutex.Unlock()
						continue
					}
					// DIMEODS-1262 - Handle status codes that can be returned from this call, namely StatusPartialContent due to byte range performed
					sc := res.StatusCode
					statusGood := sc == http.StatusOK || sc == http.StatusPartialContent || sc == http.StatusNotModified
					if res != nil && statusGood {
						return res.Body, nil
					} else {
						// DIMEODS-1262 - read and close response that we aren't going to use to avoid file leak
						if res != nil {
							if res.Body != nil {
								ioutil.ReadAll(res.Body)
								res.Body.Close()
							}
						}
					}
				}
				if err != nil {
					logger.Error("p2p cannot connect to peer (failed new request)",
						zap.Error(err),
						zap.String("peerKey", peerKey),
						zap.String("peerCN", config.GetEnvOrDefault(config.OD_PEER_CN, "")),
						zap.String("peer.CA", peer.CA),
						zap.String("peer.Cert", peer.Cert),
					)
				}
			}
		}
	}
	return nil, nil
}
