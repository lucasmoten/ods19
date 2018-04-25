package ciphertext

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/config"

	"sync"
)

var (
	// PeerSignifier indicates an identifer to use for identifying that the request is from a peer instance. It should be a value
	// that will not be presented as the user id from normal PKI connections which are distinguished names.
	// Leave this alone!  We are blocking direct access to this endpoing by setting it to something that can't be a DN.
	// It has to be the same for all peers.  If we needed it, real identifier is the cert DN which is set on
	// the user context for other values.  It CANNOT be associated with a particular user, because background processes will
	// do this on behalf of nobody in particular.
	PeerSignifier = config.GetEnvOrDefault(config.OD_PEER_SIGNIFIER, "P2P")
	// When we connect p2p, we may need to set the CN being expected
	peerCN = config.GetEnvOrDefault(config.OD_PEER_CN, "")
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

// ScheduleSetPeers sets a new peer set - there is only one thread that calls this
func ScheduleSetPeers(newPeerMap map[string]*PeerMapData) {
	setPeers(newPeerMap, peerMap)
}

// setPeers calculates which connections can be deleted an sets the new peermap
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

// UseLocalFile returns a handle to either the .cached file or .uploaded file
//  It is the caller's responsibility to close the file handle
func UseLocalFile(logger *zap.Logger, d CiphertextCache, rName FileId, cipherStartAt int64) (*os.File, int64, error) {
	var cipherFile *os.File
	var err error
	var length int64

	cipherFilePathUploaded := d.Resolve(NewFileName(rName, ".uploaded"))
	cipherFilePathCached := d.Resolve(NewFileName(rName, ".cached"))

	//Try the uploaded file
	info, ierr := d.Files().Stat(cipherFilePathUploaded)
	if ierr == nil {
		length = info.Size() - cipherStartAt
	}
	cipherFile, err = d.Files().Open(cipherFilePathUploaded)
	if err != nil {
		//Try the cached file
		info, ierr := d.Files().Stat(cipherFilePathCached)
		if ierr == nil {
			length = info.Size() - cipherStartAt
		}
		cipherFile, err = d.Files().Open(cipherFilePathCached)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, -1, nil
			}
			return nil, -1, err
		}
	}
	//We have a file handle.  Seek to where we should start reading the cipher.
	_, err = cipherFile.Seek(cipherStartAt, 0)
	if err != nil {
		logger.Error("useLocalFile failed to seek", zap.Int64("cipherStartAt", cipherStartAt))
		cipherFile.Close()
		return nil, -1, err
	}
	//Update the timestamps to note the last time it was used
	// This is done here, as well as successful end just in case of failures midstream.
	tm := time.Now()
	d.Files().Chtimes(cipherFilePathCached, tm, tm)

	return cipherFile, length, nil
}

// useP2PFile is similar to useLocalFile, except it searches peer caches for our ciphertext.
// It is better to do this than it is to stall while the file moves to S3.
//
// It is the CALLER's responsibility to close io.ReadCloser !!
func useP2PFile(logger *zap.Logger, zone CiphertextCacheZone, rName FileId, begin int64) (io.ReadCloser, error) {
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
					//P2P does not pass through nginx, so only this value can happen P2P, and we use
					//2 way SSL to enforce only peers connecting to us.
					req.Header.Set("USER_DN", PeerSignifier)
					res, err := connectionMap[peerKey].Do(req)
					if err == nil && res != nil && res.StatusCode == http.StatusOK {
						return res.Body, nil
					}
				}
				if err != nil {
					logger.Error("p2p cannot connect to peer", zap.Error(err))
				}
			}
		}
	}
	return nil, nil
}
