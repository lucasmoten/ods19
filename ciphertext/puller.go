package ciphertext

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/config"
)

const (
	pullFromUnknown = 0
	pullFromDisk    = 1
	pullFromStorage = 2
	pullFromPeer    = 3
)

// Puller is a virtual io.ReadCloser that gets range-requested chunks out of S3 to look like one contiguous file.
// It hides the range requesting out of PermanentStorage, to look like a file handle that just keeps giving back data.
//
// Note: periodically re-visit the design of this, as a proper way to range request a stream out of S3
// would allow us to simplify the design and not have to chunk the pull at all.  We chunk the pull because
// S3 will hold the entire chunk in memory after stalling until it gets that chunk.  A change to the S3 API
// might let us simplify a lot of things:
//
// https://github.com/aws/aws-sdk-go/issues/915
//
type Puller struct {
	CiphertextCache *CiphertextCacheData `json:"-"`
	Logger          *zap.Logger          `json:"-"`
	// TotalLen is total size of this range request - if known, determined by CipherStart,CipherStop
	TotalLen int64
	// CipherStart must be cipher block aligned... not necessarily what browser asked for (may be slightly lower)
	CipherStart int64
	// CipherStop is one less than block align. It might be -1 if there is no known end
	CipherStop int64
	// Key is the same as an S3 bucket key.  It contains the partition and the RName
	Key *string `json:"-"`
	// RName is a randomly generated identifier for this version of the file
	RName FileId
	// File the current reader from which we are drawing data for the range request (or a subset of it)
	File io.ReadCloser `json:"-"`
	// Remaining bytes unread in File
	Remaining int64
	// ChunkSize is the max we will pull into a file for PermanentStorage
	ChunkSize int64
	// Index is the location within File (not the range request)
	Index int64
	// IsLocal lets caller know if this came from a file
	IsLocal bool
	// IsP2P lets caller know that we should try P2P -- because we previously got a chunk from it
	IsP2P bool
	// From lets the caller know exactly where File was populated from using the constants above.
	// We use it to set a large chunk for non-PermanentStorage to make range requesting not pointlessly scrap and
	// get a new File every 16MB; which is unfortunately necessary with s3manager due to holding things in memory.
	From int
}

// NewPuller prepares to start pulling ciphertexts.  This should now be the ONLY way to get them.
func (d *CiphertextCacheData) NewPuller(logger *zap.Logger, rName FileId, totalLength, cipherStartAt, cipherStopAt int64) (io.ReadCloser, bool, error) {
	key := toKey(string(d.Resolve(NewFileName(rName, ""))))
	p := &Puller{
		CiphertextCache: d,
		Logger:          logger,
		TotalLen:        totalLength,
		CipherStart:     cipherStartAt,
		CipherStop:      cipherStopAt,
		Key:             key,
		RName:           rName,
		File:            nil,
		ChunkSize:       d.ChunkSize,
		Index:           cipherStartAt,
	}

	//look in local and permanent storage first
	err := p.More(false)
	if err != nil {
		// DIMEODS-1262 - additional logging
		logger.Debug(
			"PermanentStorage pull attempt failed. Will try from peer if enabled, or stall waiting for peer to upload",
			zap.String("od_peer_enabled", os.Getenv(config.OD_PEER_ENABLED)),
			zap.String("error", err.Error()),
		)
		if strings.ToLower(os.Getenv(config.OD_PEER_ENABLED)) == "true" {
			//This will find it in a peer if it wasn't already found
			err = p.More(true)
		}
	}
	sleepTime := time.Duration(1) * time.Second
	attempts := 20
	for attempts > 0 && err != nil && d.PermanentStorage != nil {
		//Keep trying PermanentStorage
		//This can happen due to temporarily losing the only node that has the ciphertext. Maybe we should just get an error in this case.
		err = p.More(false)
		if err == nil {
			break
		}
		attempts--
		if attempts == 0 {
			// DIMEODS-1262 - additional logging to relay what happened
			logger.Debug(
				"PermanentStorage pull stall",
				zap.Int("attempts", int(attempts)),
			)
			break
		}
		sleepTime = sleepTime + sleepTime
		// DIMEODS-1262 - logging level state and added attempts remaining
		logger.Debug(
			"PermanentStorage pull stall",
			zap.Int("attempts", int(attempts)),
			zap.Int("sleepInSeconds", int(sleepTime/time.Second)),
		)
		time.Sleep(sleepTime)
	}
	if err != nil {
		// DIMEODS-1262 - additional logging to relay what happened
		logger.Warn(
			"PermanentStorage pull stall",
			zap.String("error", err.Error()),
		)
		return nil, false, err
	}
	return p, p.IsLocal, nil
}

// getFileHandle will try to get a file handle from the best location.
// end is only used for GetStream pulls, which have high latency because we cannot stream until we have the file.
func (p *Puller) getFileHandle(begin, end int64, p2p bool) (io.ReadCloser, error) {
	// Always check disk first - this lets us switch to disk when background cache finishes.
	file, _, err := UseLocalFile(p.Logger, p.CiphertextCache, p.RName, begin)
	if file != nil {
		p.Logger.Debug("puller getting file locally")
		p.IsLocal = true
		p.From = pullFromDisk
		return file, nil
	} else {
		p.Logger.Debug("puller didn't find file locally, will try p2p or PermanentStorage")
	}
	// try p2p if asked, or if no PermanentStorage even exists
	p.IsLocal = false
	var fileP2P io.ReadCloser
	// Having no permanent storage is like an implicit p2p flag
	if p2p || p.IsP2P || p.CiphertextCache.GetPermanentStorage() == nil {
		p.Logger.Debug("puller will try p2p")
		fileP2P, err = useP2PFile(p.Logger, p.CiphertextCache.CiphertextCacheZone, p.RName, begin)
		if err != nil {
			p.Logger.Info("puller cant use p2p", zap.Error(err))
		}
	}
	if fileP2P != nil {
		p.Logger.Debug("puller getting file from p2p")
		// If we got a chunk p2p, then we need to be allowed to continue to run p2p for the remainder of this pull
		p.IsP2P = true
		p.From = pullFromPeer
		return fileP2P, nil
	}
	if p.CiphertextCache.GetPermanentStorage() == nil {
		// We are doomed to lose connection here.  It will get logged.
		return nil, fmt.Errorf("puller did not use p2p and we have no PermanentStorage")
	}
	// Range request it out of PermanentStorage if we can
	p.Logger.Debug("puller will try to range request from PermanentStorage", zap.String("key", *p.Key), zap.Int64("begin", begin), zap.Int64("end", end))
	f, err := p.CiphertextCache.GetPermanentStorage().GetStream(p.Key, begin, end)
	p.Logger.Debug("puller getting file from PermanentStorage")
	if err == nil && f != nil {
		p.From = pullFromStorage
	} else {
		// We are doomed to lose the connection.  It will get logged.
		p.Logger.Warn("puller is doomed to lose connection")
		if f != nil {
			// DIMEODS-1262 - is this a potential file leak?
			p.Logger.Warn("may have potential file leak in puller.go 174")
		}
		p.From = pullFromUnknown
	}
	return f, err
}

// More will refresh from PermanentStorage for more data
func (p *Puller) More(useP2P bool) error {
	if p.Index == p.TotalLen {
		return io.EOF
	}
	begin := p.Index
	end := int64(-1)

	// Compute the indices that we need for getting this range
	cLength := p.ChunkSize
	if p.Index+cLength < p.TotalLen {
		// end is only used if PermanentStorage was chosen - because we can't just stream huge files with s3manager.
		end = p.Index + cLength - 1
	} else {
		cLength = p.TotalLen - p.Index
	}

	f, err := p.getFileHandle(begin, end, useP2P)
	p.Logger.Debug("puller returned from getting file handle")
	if p.From != pullFromStorage {
		// Recompute for at least a 1GB chunk len to avoid
		// open/close/search every 16MB - bad performance on large file downloads
		cLength = int64(1024 * 1024 * 1024)
		if p.Index+cLength < p.TotalLen {
			// this is computed for consistency, but note that we only reference cLength after this.
			end = p.Index + cLength - 1
		} else {
			cLength = p.TotalLen - p.Index
		}
	}

	if err != nil {
		p.Logger.Error(
			"unable to get a puller filehandle",
			zap.String("key", *p.Key),
			zap.Error(err),
		)
		return err
	}
	if f == nil {
		p.Logger.Error("puller cannot get filehandle")
	}
	p.File = f
	p.Remaining = cLength
	return nil
}

// Keep calling this to read out of PermanentStorage
func (p *Puller) Read(data []byte) (int, error) {
	var err error
	var length int
	if p.Remaining > 0 {
		length, err = p.File.Read(data)
		p.Remaining -= int64(length)
		p.Index += int64(length)
	}
	if p.Remaining == 0 {
		err = p.More(false)
	}
	return length, err
}

// Close this when done pulling from PermanentStorage
func (p *Puller) Close() error {
	if p.File != nil {
		//Close out this chunk from PermanentStorage
		return p.File.Close()
	}
	return nil
}
