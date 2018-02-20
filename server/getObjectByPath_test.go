package server_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/util"
)

func verifyfilepath(t *testing.T, clientid int, path string) {
	uri1 := mountPoint + path
	req1, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		t.Errorf("Error creating request")
	}
	res1, _ := clients[clientid].Client.Do(req1)
	defer util.FinishBody(res1.Body)
	if res1.StatusCode != 200 {
		t.Errorf("bad status: expected 200, but got %v for %s", res1.StatusCode, path)
	}
}

func TestGetObjectByPathForUser(t *testing.T) {
	tester10 := 0

	// setup objects
	// The html file
	cor1 := protocol.CreateObjectRequest{
		Name:        "TestGetObjectsByPathForUser.html",
		RawAcm:      `{"version":"2.1.0","classif":"U"}`,
		ContentType: "text/html",
		TypeName:    "File",
	}
	f1 := bytes.NewBuffer([]byte(`<html><head><title>TestGetObjectsByPathForUser</title></head><body>This is a test. There should be an animated image displayed here<img src="animated.gif" /></body></html>`))
	clients[tester10].C.CreateObject(cor1, f1)
	// The linked image
	imageurl := "https://media.giphy.com/media/VlzUkJJzvz0UU/giphy.gif"
	cor2 := protocol.CreateObjectRequest{
		Name:        "animated.gif",
		RawAcm:      `{"version":"2.1.0","classif":"U"}`,
		ContentType: "image/gif",
		TypeName:    "File",
	}
	f2, e := http.Get(imageurl)
	if e != nil {
		t.Errorf("cant retrieve image %s - %s", imageurl, e.Error())
	}
	defer f2.Body.Close()
	clients[tester10].C.CreateObject(cor2, f2.Body)

	// attempt to retrieve by filenames
	verifyfilepath(t, tester10, "/files/"+cor1.Name)
	verifyfilepath(t, tester10, "/files/"+cor2.Name)
}

func TestGetObjectByPathForGroup(t *testing.T) {
	tester10 := 0

	// setup objects

	// The html file
	cor1 := protocol.CreateObjectRequest{
		Name:        "TestGetObjectsByPathForGroup.html",
		RawAcm:      `{"version":"2.1.0","classif":"U"}`,
		ContentType: "text/html",
		TypeName:    "File",
		OwnedBy:     "group/dctc/odrive",
	}
	pageContent := `
<html>
<head><title>TestGetObjectsByPathForGroup</title></head>
<body>
<h2>Data</h2>
<h3>Secure and trusted storage</h3>
<p>
Store, protect, share, and process your data in a personal, secure, controlled space.
</p>
<p>
Data provides API, filesystem, and encryption layers atop pluggable storage backends, 
such as Amazon AWS S3. Encryption keys are stored such that the compromise of a single
machine is insufficient to decrypt any data, compromises cannot spread between objects, 
users never have direct possession of object keys, and yet authorized emergency decryption 
remains possible. Sharing is cryptographically enforced. Data serves as the data hub of 
Grey Matter.
</p>
<img src="diagram-gm-data.png" />
<hr />
<video controls autoplay name="media">
  <source src="spacexvid.mp4" type="video/mp4">
</video>
</body></html>`
	f1 := bytes.NewBuffer([]byte(pageContent))
	clients[tester10].C.CreateObject(cor1, f1)

	// The linked image
	imageurl := "http://deciphernow.com/img/diagrams/diagram-gm-data.png"
	cor2 := protocol.CreateObjectRequest{
		Name:        "diagram-gm-data.png",
		RawAcm:      `{"version":"2.1.0","classif":"U"}`,
		ContentType: "image/png",
		TypeName:    "File",
		OwnedBy:     "group/dctc/odrive",
	}
	f2, e := http.Get(imageurl)
	if e != nil {
		t.Errorf("cant retrieve image %s - %s", imageurl, e.Error())
	}
	defer f2.Body.Close()
	clients[tester10].C.CreateObject(cor2, f2.Body)

	// How about an MP4
	mp4url := "https://r1---sn-ab5sznld.googlevideo.com/videoplayback?ei=Yw6JWpqUEobA8wTj-pDIDg&key=yt6&initcwndbps=1646250&expire=1518953155&pl=19&itag=18&mt=1518931480&ms=au,rdu&requiressl=yes&mime=video/mp4&dur=168.483&id=o-AGVenT0xptAyMOBz4kRgbMFsYBDBn_cVtJehuWNWlc6v&c=WEB&mn=sn-ab5sznld,sn-ab5l6n67&mm=31,29&ip=209.205.200.210&sparams=clen,dur,ei,gir,id,initcwndbps,ip,ipbits,itag,lmt,mime,mm,mn,ms,mv,pl,ratebypass,requiressl,source,expire&mv=m&lmt=1517957677303776&ipbits=0&gir=yes&ratebypass=yes&fvip=1&clen=7757173&signature=BFA0D4FED9649AFC52097A81835E32E5C025007B.9A8A764872B3AFA62EACF85EB272EC5FC099AC43&source=youtube&type=video%2Fmp4%3B+codecs%3D%22avc1.42001E%2C+mp4a.40.2%22&quality=medium&title=(Hdvidz.in)_SpaceX-Launches-The-Falcon-Heavy-Rocket--Why-Its-Such-A-Big-Deal-For-Elon-Musk--TIME"
	cor3 := protocol.CreateObjectRequest{
		Name:        "spacexvid.mp4",
		RawAcm:      `{"version":"2.1.0","classif":"U"}`,
		ContentType: "video/mp4",
		TypeName:    "File",
		OwnedBy:     "group/dctc/odrive",
	}
	f3, e := http.Get(mp4url)
	if e != nil {
		t.Errorf("cant retrieve image %s - %s", mp4url, e.Error())
	}
	defer f3.Body.Close()
	clients[tester10].C.CreateObject(cor3, f3.Body)

	// attempt to retrieve by filenames
	verifyfilepath(t, tester10, "/files/groupobjects/dctc_odrive/"+cor1.Name)
	verifyfilepath(t, tester10, "/files/groupobjects/dctc_odrive/"+cor2.Name)
	verifyfilepath(t, tester10, "/files/groupobjects/dctc_odrive/"+cor3.Name)
}
