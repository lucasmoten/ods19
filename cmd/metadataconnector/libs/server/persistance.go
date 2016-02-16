package server

import "io"

// Persistance should be implented by durable persistence engines, such as S3.
type Persistance interface {
	Put(r io.Reader, id string) error
	Get(id string)
	Delete(id string)
	List(name string)
}

// TODO create constructor for Persistance
// func NewS3Persistance(session *aws.Session) (Persistance, error) {
//
// 	return
// }
