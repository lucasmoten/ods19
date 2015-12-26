package libs

import (
	"io"
)

// StatsNeeded We send one of these when we want to query statistics
type StatsNeeded struct {
}

// Stat is some number of events over an observation period
type Stat struct {
	EventType  string
	BeginTime  int64
	EndTime    int64
	EventCount int64
}

// StatCollect is where we aggregate statistical reports for event types
type StatCollect struct {
	EventCount        int64
	ObservationPeriod int64
	Units             string
	Name              string
}

/*Uploader is a special type of Http server.
  Put any config state in here.
  The point of this server is to show how
  upload and download can be extremely efficient
  for large files.
*/
type Uploader struct {
	Partition      string
	Port           int
	Bind           string
	Addr           string
	UploadCookie   string
	BufferSize     int
	KeyBytes       int
	RSAEncryptBits int
	Backend        *Backend
	StatsReport    chan Stat
	StatsNeeded    chan StatsNeeded
	StatsQuery     chan []StatCollect
}

//Backend can be implemented as S3, filesystem, etc
type Backend struct {
	GetReadHandle         func(fileName string) (r io.Reader, c io.Closer, err error)
	GetWriteHandle        func(fileName string) (w io.Writer, c io.Closer, err error)
	EnsurePartitionExists func(fileName string) error
	GetFileExists         func(fileName string) (bool, error)
	GetAppendHandle       func(fileName string) (w io.Writer, c io.Closer, err error)
}
