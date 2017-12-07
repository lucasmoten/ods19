package server_test

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"testing"
)

/**
  When testing, in order to log traffic nicely:

  //Open up a CreateObject.html file of actual traffic
  tl := NewTrafficLog("CreateObject")
  ...
  //Mark the request with a description of what we are doing
  tl.Request(t, req,
    &TrafficLogDescription{
       OperationName: "Make an object",
       RequestDescription: `
           This POST request is multi-part mime, and
           places an object into odrive so we can get back an id ....
       `,
       ResponseDescription: `
           This json gives us the id, and ....
       `,
    },
  )
  ...
  //Mark the response so that the data is collected
  tl.Response(t, res)

  //Multiple examples can exist in one file...
  tl.Request(...)
  ...
  tl.Response(...)
*/

//TrafficLog is the state required to do request logs

type TrafficLog struct {
	File         *os.File
	Name         string
	Descriptions []*TrafficLogDescription
	Current      *TrafficLogDescription
}

//TrafficLogDescription is passed in to ensure that an operation is correctly described in logs
type TrafficLogDescription struct {
	OperationName       string
	RequestDescription  string
	ResponseDescription string
	RequestBodyHide     bool
	ResponseBodyHide    bool
	RequestBytes        []byte
	ResponseBytes       []byte
	OpHash              uint32
}

//NewTrafficLog creates a logging context for dumping requests
func NewTrafficLog(testName string) *TrafficLog {
	return &TrafficLog{
		Name:         testName,
		Descriptions: make([]*TrafficLogDescription, 0),
	}
}

var requestLoggingStyle = `
    body {
        font-family: "Arial", Times, serif;
    }
    pre.request {
        background-color: black;
        color: rgb(0,255,0);
        padding: 20px 20px 20px 20px;
        border-radius: 10px;
    }
    pre.response {
        background-color: black;
        color: rgb(64,255,64);
        padding: 20px 20px 20px 20px;
        border-radius: 10px;
    }
	span.notshown {
		color: red;
	}
`

func (trafficLog *TrafficLog) render() {
	// Header
	gopath := os.Getenv("GOPATH")
	if len(gopath) == 0 {
		log.Printf("GOPATH needs to be set for request logging to work")
	}
	fqRoot := fmt.Sprintf("%s/src/github.com/deciphernow/object-drive-server", gopath)
	fqName := fmt.Sprintf("%s/server/static/templates/%s.html", fqRoot, trafficLog.Name)
	var err error
	trafficLog.File, err = os.Create(fqName)
	if err != nil {
		log.Printf("Unable to open %s: %v", fqName, err)
		return
	}
	defer trafficLog.File.Close()

	trafficLog.File.WriteString("<html>\n")
	trafficLog.File.WriteString(fmt.Sprintf("<head><title>%s</title>\n", trafficLog.Name))
	trafficLog.File.WriteString(fmt.Sprintf("<style>\n%s\n</style>\n", requestLoggingStyle))
	trafficLog.File.WriteString("</head>\n")
	trafficLog.File.WriteString("<body>\n")
	trafficLog.File.WriteString(fmt.Sprintf("<h1>%s</h1>\n", trafficLog.Name))
	trafficLog.File.WriteString(`
			This is a sampling of successful tests that exhibit correct cases for using the API,
			at the level of http traffic.  
		`)
	trafficLog.File.WriteString("<h2>Table Of Contents</h2>")
	trafficLog.File.WriteString("<ul>")
	for _, d := range trafficLog.Descriptions {
		trafficLog.File.WriteString(fmt.Sprintf("<li><a href=\"#%d\">%s</a></li>", d.OpHash, d.OperationName))
	}
	trafficLog.File.WriteString("</ul>")
	trafficLog.File.WriteString("<br/>")

	for _, d := range trafficLog.Descriptions {
		trafficLog.File.WriteString(fmt.Sprintf("<h2 id=%d>%s</h2>\n", d.OpHash, d.OperationName))
		trafficLog.File.WriteString(fmt.Sprintf("%s\n", d.RequestDescription))
		trafficLog.File.WriteString("<pre class=\"request\">\n")

		trafficLog.File.Write(d.RequestBytes)
		if d.RequestBodyHide {
			trafficLog.File.WriteString("<span class=\"notshown\">....</span>\n")
		}
		trafficLog.File.WriteString("</pre>\n")

		if len(d.ResponseBytes) > 0 {
			trafficLog.File.WriteString(fmt.Sprintf("%s\n", d.ResponseDescription))
			trafficLog.File.WriteString("<pre class=\"response\">\n")
			trafficLog.File.Write(d.ResponseBytes)
			if d.ResponseBodyHide {
				trafficLog.File.WriteString("<span class=\"notshown\">....</span>\n")
			}
			trafficLog.File.WriteString("</pre>\n")
		}
	}

	trafficLog.File.WriteString("</body></html>\n")
}

//Request ensures that we are actually logging to a file
func (trafficLog *TrafficLog) Request(t *testing.T, r *http.Request, description *TrafficLogDescription) {
	trafficLog.Current = description
	if description == nil {
		return
	}

	bytes, err := httputil.DumpRequest(r, !description.RequestBodyHide)
	if err != nil {
		log.Printf("Cannot log request: %v", err)
		return
	}
	trafficLog.Descriptions = append(trafficLog.Descriptions, description)
	description.RequestBytes = bytes
	h := fnv.New32a()
	h.Write([]byte(description.OperationName))
	description.OpHash = h.Sum32()

	return
}

//Response will dump the response if we are actually doing logging, and log the message anyway
func (trafficLog *TrafficLog) Response(t *testing.T, w *http.Response) {
	if trafficLog.Current == nil {
		return
	}

	bytes, err := httputil.DumpResponse(w, !trafficLog.Current.RequestBodyHide)
	if err != nil {
		log.Printf("failed to dump response: %v", err)
	}
	trafficLog.Current.ResponseBytes = bytes

}

//Close if we are actually writing to the log
func (trafficLog *TrafficLog) Close() {
	trafficLog.render()
}
