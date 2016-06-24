package logparser

import "github.com/boltdb/bolt"

type Collector struct {
	DB *bolt.DB
}
