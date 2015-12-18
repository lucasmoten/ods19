package libs

type fileDirPath string
type bindIPAddr string
type bindURL string

/*Uploader is a special type of Http server.
  Put any config state in here.
  The point of this server is to show how
  upload and download can be extremely efficient
  for large files.
*/
type Uploader struct {
	HomeBucket     fileDirPath
	Port           int
	Bind           bindIPAddr
	Addr           bindURL
	UploadCookie   string
	BufferSize     int
	KeyBytes       int
	RSAEncryptBits int
}
