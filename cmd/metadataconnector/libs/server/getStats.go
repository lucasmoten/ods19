package server

import (
	"decipher.com/oduploader/performance"
	"fmt"
	"net/http"
)

func (h AppServer) getStats(w http.ResponseWriter, r *http.Request, caller Caller) {
	fmt.Fprintf(w, "\nUploaders Aggregate:\n")
	h.Tracker.Reporters[performance.UploadCounter].Q.Dump(w)

	fmt.Fprintf(w, "\nDownloaders Aggregate:\n")
	h.Tracker.Reporters[performance.DownloadCounter].Q.Dump(w)

}
