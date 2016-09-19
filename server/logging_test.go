package server_test

import (
	"fmt"
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
	File        *os.File
	Name        string
	Description *TrafficLogDescription
}

//TrafficLogDescription is passed in to ensure that an operation is correctly described in logs
type TrafficLogDescription struct {
	OperationName       string
	RequestDescription  string
	ResponseDescription string
	RequestBodyHide     bool
	ResponseBodyHide    bool
}

//NewTrafficLog creates a logging context for dumping requests
func NewTrafficLog(testName string) *TrafficLog {
	return &TrafficLog{
		Name: testName,
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

//Request ensures that we are actually logging to a file
func (trafficLog *TrafficLog) Request(t *testing.T, r *http.Request, description *TrafficLogDescription) {
	if description == nil {
		return
	}

	t.Log(fmt.Sprintf("%s request", description.OperationName))

	if trafficLog.File == nil {
		//First-time setup
		gopath := os.Getenv("GOPATH")
		if len(gopath) == 0 {
			log.Printf("GOPATH needs to be set for request logging to work")
		}
		fqRoot := fmt.Sprintf("%s/src/decipher.com/object-drive-server", gopath)
		fqName := fmt.Sprintf("%s/server/static/templates/%s.html", fqRoot, trafficLog.Name)
		var err error
		trafficLog.File, err = os.Create(fqName)
		if err != nil {
			log.Printf("Unable to open %s: %v", fqName, err)
			return
		}
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
	} else {
		trafficLog.File.WriteString("<hr/>")
	}
	trafficLog.Description = description

	trafficLog.File.WriteString(fmt.Sprintf("<h2>%s</h2>\n", trafficLog.Description.OperationName))
	trafficLog.File.WriteString(fmt.Sprintf("%s\n", trafficLog.Description.RequestDescription))
	trafficLog.File.WriteString("<pre class=\"request\">\n")
	rData, err := httputil.DumpRequest(r, !trafficLog.Description.RequestBodyHide)
	if err != nil {
		log.Printf("Unable to dump request %s: %v", trafficLog.Description.OperationName, err)
		return
	}

	trafficLog.File.Write(rData)
	if trafficLog.Description.RequestBodyHide {
		trafficLog.File.WriteString("<span class=\"notshown\">....</span>\n")
	}
	trafficLog.File.WriteString("</pre>\n")
}

//Response will dump the response if we are actually doing logging, and log the message anyway
func (trafficLog *TrafficLog) Response(t *testing.T, w *http.Response) {
	if trafficLog.Description == nil {
		return
	}

	t.Log(fmt.Sprintf("%s response", trafficLog.Description.OperationName))

	trafficLog.File.WriteString(fmt.Sprintf("%s\n", trafficLog.Description.ResponseDescription))
	trafficLog.File.WriteString("<pre class=\"response\">\n")
	rData, err := httputil.DumpResponse(w, !trafficLog.Description.ResponseBodyHide)

	if err != nil {
		log.Printf("Unable to dump request %s: %v", trafficLog.Description.OperationName, err)
	}
	trafficLog.File.Write(rData)
	if trafficLog.Description.ResponseBodyHide {
		trafficLog.File.WriteString("<span class=\"notshown\">....</span>\n")
	}
	trafficLog.File.WriteString("</pre>\n")
}

//Close if we are actually writing to the log
func (trafficLog *TrafficLog) Close() {
	if trafficLog.File == nil {
		return
	}
	trafficLog.File.WriteString("</body></html>\n")
	trafficLog.File.Close()
	trafficLog.File = nil
}
