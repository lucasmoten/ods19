package server

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"time"
)

/*CacheMustExist ensures that the cache directory exists
 */
func (h AppServer) CacheMustExist() (err error) {
	if _, err = os.Stat(h.CacheLocation); os.IsNotExist(err) {
		err = os.Mkdir(h.CacheLocation, 0700)
		log.Printf("Creating cache directory %s", h.CacheLocation)
		if err != nil {
			log.Printf("Cannot create cache directory: %v", err)
			return err
		}
	}
	return err
}

/*fileVisit visits every file in the cache to see if we should delete it
 */
func fileVisit(name string, f os.FileInfo, err error) (errReturn error) {

	//Ignore directories.  We should not have an unbounded number of directories.
	//And we must ignore h.CacheLocation
	if f.IsDir() {
		return nil
	}

	t := f.ModTime().Unix() //In units of second
	n := time.Now().Unix()  //In unites of second
	ageInSeconds := n - t
	size := f.Size()
	ext := path.Ext(name)

	switch {
	case ext == ".cached":
		/**
						  Simple purging scheme:
						    a file must be older than a minute since last use
						    its (integer) value is its size divided by age squared
						    when its value is zero, get rid of it.

						    this does NOT take into account the available space,
						    nor the insert rate.  it is rather agressive though.
				        Example values:
				        10GB 1day - 1
				         6GB 1day - 0
				       500MB 4hrs - 2
		             1MB 15mins - 1
		             1MB 20mins - 0
		*/
		if ageInSeconds > 60 {
			value := size / (ageInSeconds * ageInSeconds)
			if value == 0 {
				errReturn := os.Remove(name)
				if errReturn != nil {
					log.Printf("Unable to purge %s", name)
				} else {
					log.Printf("Purged %s.  Age:%ds Size:%d", name, ageInSeconds, size)
				}
			}
		}
	default:
		//If something has been here for a week, and it's not cached, then it's
		//garbage.  If a machine has been turned off for a few days, the files
		//might legitimately be awaiting upload.  Other states are certainly
		//garbage after only a few hours.
		if ageInSeconds > 60*60*24*7 {
			errReturn := os.Remove(name)
			if errReturn != nil {
				log.Printf("Unable to purge %s", name)
			} else {
				log.Printf("Purged %s.  Age:%ds Size:%d", name, ageInSeconds, size)
			}
		}
	}
	return
}

/*CachePurge will periodically delete files that do not need to be in the cache.
 */
func (h AppServer) CachePurge() {
	err := h.CacheMustExist()
	if err == nil {
		for {
			err := filepath.Walk(h.CacheLocation, fileVisit)
			if err != nil {
				log.Printf("Unable to walk cache %s: %v", h.CacheLocation, err)
			}
			time.Sleep(30 * time.Second)
		}
	}
}
