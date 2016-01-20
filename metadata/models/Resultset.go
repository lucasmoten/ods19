package models

type Resultset struct {
	TotalRows  int
	PageCount  int
	PageNumber int
	PageSize   int
	PageRows   int
}
