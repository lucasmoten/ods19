
# Clients

Besides the odrive UI that will be at:  `https://gatekeeper:8080/apps/drive/index.html` once containers start, there is a client library available.  The client can be used directly, as is called in the 
Fetch function described here.  Or it is generally going to be more useful to call the client
library in response to events (modifications to files).  

The primary use case is to hear that a pdf or image was uploaded, and to respond by creating an OCR, text extract, or caption of this file.
When more files are posted in response, then things like sentiment analysis and language translation
can proceed.

```go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/client"
	"bitbucket.di2e.net/dime/object-drive-server/events"
)

func FetchStream(c *client.OdriveResponder, odc *client.Client, gem *events.GEM) error {
	userDn := gem.Payload.UserDN
	objectId := gem.Payload.ObjectID

	// Diagnostics
	url := fmt.Sprintf("%s/objects/%s", c.Conf.Remote, objectId)
	properties := fmt.Sprintf("%s/stream", url)
	c.Note("  fetch")
	c.Note("    url: %s", properties)
	c.Note("    as:  %s", userDn)

	// API is wrong here.  We need an io.ReadCloser in order to partially consume streams
	rdr, err := odc.GetObjectStream(objectId)
	if err != nil {
		return err
	}
	defer rdr.Close()
	c.Note("  fetch")
	c.Note("    url: %s", properties)
	c.Note("    as:  %s", userDn)

	return nil
}

func Fetch(c *client.OdriveResponder, gem *events.GEM) (bool, error) {
	if gem.Action == "create" || gem.Action == "update" || gem.Action == "move" {
		if gem.Payload.ContentType != "" && gem.Payload.ContentSize > 0 {
			odc, err := client.NewClient(c.Conf)
			if err != nil {
				return true, err
			}
			odc.MyDN = gem.Payload.ObjectID
			err = FetchStream(c, odc, gem)
			if err != nil {
				return true, nil
			}
		}
	}
	return true, nil
}

func main() {
	cert := os.Args[1]
	trust := os.Args[2]
	key := os.Args[3]
	remote := os.Args[4]
	serverName := os.Args[5]
	group := os.Args[6]
	zk := os.Args[7]
	conf := client.Config{
		Cert:       cert,
		Trust:      trust,
		Key:        key,
		SkipVerify: false,
		Remote:     remote,
		ServerName: serverName,
	}
	c, err := client.NewOdriveResponder(conf, group, zk, Fetch)
	if err != nil {
		log.Printf("error creating: %v", err)
		os.Exit(-1)
	}
	c.DebugMode = true
	c.Note("consume kafka for a few minutes")
	go func() {
		for {
			err = c.ConsumeKafka()
			if err != nil {
				log.Printf("error connecting: %v", err)
				os.Exit(-1)
			}
		}
	}()
	time.Sleep(5 * time.Minute)
	os.Exit(0)
}
```
