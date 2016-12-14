package server_test

import (
	"mime"
	"testing"

	"decipher.com/object-drive-server/server"
)

func TestGetContentTypeFromFilename(t *testing.T) {

	expected := make(map[string]string)
	expected["../watever"] = "application/octet-stream"
	expected["this is the file.txt"] = "text/plain"
	expected[""] = "application/octet-stream"
	expected["somereport.doc"] = "application/msword"
	expected["helicopters.wmv"] = "video/x-ms-wmv"

	for filename, expectedContentType := range expected {
		actualContentType := server.GetContentTypeFromFilename(filename)
		if actualContentType != expectedContentType {
			t.Logf("filename '%s' didnt produce expected content type '%s'. Actual type is %s", filename, expectedContentType, actualContentType)
			t.Fail()
		}
	}
}

// This test wont fail, but will show what the mime package TypeByExtension will report
// This requires /etc/mime.types or other files to be set.
func TestGetContentTypeFromFilenameVsMimeTypeByExtension(t *testing.T) {
	t.Logf("%20s %10s %40s %40s", "Filename", "Ext", "mime.TypeByExtension()", "our map")
	for k, v := range server.ExtensionToContentType {
		filename := "file." + k
		mpv := mime.TypeByExtension(filename)
		spv := v
		if spv != mpv {
			t.Logf("%20s %10s %40s %40s'", filename, k, mpv, spv)
		}
	}
}
