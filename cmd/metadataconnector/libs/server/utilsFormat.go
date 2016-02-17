package server

import "time"

// getFormattedDate formats a passed in time as RFC3339 format, which is
// basically:    YYYY-MM-DDTHH:mm:ss.nnnZ
// TODO: Move this utility method to a common file to make it clear its
// available by all operations
func getFormattedDate(t time.Time) string {
	return t.Format(time.RFC3339)
}
