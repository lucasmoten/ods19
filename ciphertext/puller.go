package ciphertext

import (
	"fmt"
	"io"
	"time"

	"github.com/uber-go/zap"
)

// Puller is a virtual io.ReadCloser that gets range-requested chunks out of S3 to look like one contiguous file.
// It hides the range requesting out of PermanentStorage, to look like a file handle that just keeps giving back data.
type Puller struct {
	CiphertextCache *CiphertextCacheData `json:"-"`
	Logger          zap.Logger           `json:"-"`
	TotalLen        int64
	CipherStart     int64
	CipherStop      int64
	Key             *string `json:"-"`
	RName           FileId
	File            io.ReadCloser `json:"-"`
	Remaining       int64
	ChunkSize       int64
	Index           int64
	IsLocal         bool
	IsP2P           bool
}

// NewPuller prepares to start pulling ciphertexts.  This should now be the ONLY way to get them.
func (d *CiphertextCacheData) NewPuller(logger zap.Logger, rName FileId, totalLength, cipherStartAt, cipherStopAt int64) (io.ReadCloser, bool, error) {
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

	//look in permanent storage first
	err := p.More(false)
	if err != nil {
		//This will find it in a peer if it wasn't already found
		err = p.More(true)
	}
	sleepTime := time.Duration(1) * time.Second
	attempts := 20
	for attempts > 0 && err != nil && d.PermanentStorage != nil {
		//Keep trying PermanentStorage
		//In general, we should never get here, because it's a barely bounded stall.
		err = p.More(false)
		if err == nil {
			break
		}
		attempts--
		if attempts == 0 {
			break
		}
		sleepTime = sleepTime + sleepTime
		p.Logger.Info(
			"PermanentStorage pull stall",
			zap.Int("sleepInSeconds", int(sleepTime/time.Second)),
		)
		time.Sleep(sleepTime)
	}
	if err != nil {
		return nil, false, err
	}
	return p, p.IsLocal, nil
}

// getFileHandle will try to get a file handle from the best location.
// end is only used for GetStream pulls, which have high latency because we cannot stream until we have the file.
func (p *Puller) getFileHandle(begin, end int64, p2p bool) (io.ReadCloser, error) {
	////Always check disk first - this lets us switch to disk when background cache finishes.
	file, _, err := UseLocalFile(p.CiphertextCache.Logger, p.CiphertextCache, p.RName, begin)
	if file != nil {
		p.IsLocal = true
		return file, nil
	}
	//try p2p if asked, or if no PermanentStorage even exists
	p.IsLocal = false
	var filep2p io.ReadCloser
	//Having no permanent storage is like an implicit p2p flag
	if p2p || p.IsP2P || p.CiphertextCache.GetPermanentStorage() == nil {
		filep2p, err = useP2PFile(p.Logger, p.CiphertextCache.CiphertextCacheSelector, p.RName, begin)
	}
	if err != nil {
		p.Logger.Info("puller cant use p2p", zap.String("err", err.Error()))
	}
	if filep2p != nil {
		//If we got a chunk p2p, then we need to be allowed to continue to run p2p for the remainder of this pull
		p.IsP2P = true
		return filep2p, nil
	}
	if p.CiphertextCache.GetPermanentStorage() == nil {
		return nil, fmt.Errorf("puller did not use p2p and we have no PermanentStorage")
	}
	//Range request it out of PermanentStorage if we can
	f, err := p.CiphertextCache.GetPermanentStorage().GetStream(p.Key, begin, end)
	return f, err
}

// More will refresh from PermanentStorage for more data
func (p *Puller) More(useP2P bool) error {
	//Compute the indices that we need for getting this range
	clen := p.ChunkSize
	if p.Index == p.TotalLen {
		return io.EOF
	}
	begin := p.Index
	end := int64(-1)
	//These numbers should have been snapped to cipher block boundaries if they were not already
	if p.Index+clen < p.TotalLen {
		end = p.Index + clen - 1
	} else {
		clen = p.TotalLen - p.Index
	}
	f, err := p.getFileHandle(begin, end, useP2P)
	if err != nil {
		p.Logger.Error(
			"unable to get a puller filehandle",
			zap.String("key", *p.Key),
			zap.String("err", err.Error()),
		)
		return err
	}
	if f == nil {
		p.Logger.Error("puller cannot get filehandle")
	}
	p.File = f
	p.Remaining = clen
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
