package models

// UserStatsMetrics are the raw values that get added together
type UserStatsMetrics struct {
	TypeName                string `db:"TypeName" json:"typeName"`
	Objects                 int    `db:"Objects" json:"objects"`
	ObjectsAndRevisions     int    `db:"ObjectsAndRevisions" json:"objectsAndRevisions"`
	ObjectsSize             int64  `db:"ObjectsSize" json:"objectsSize"`
	ObjectsAndRevisionsSize int64  `db:"ObjectsAndRevisionsSize" json:"objectsAndRevisionsSize"`
}

// UserStats are per user statistics
type UserStats struct {
	TotalObjects                 int
	TotalObjectsAndRevisions     int
	TotalObjectsSize             int64
	TotalObjectsAndRevisionsSize int64
	ObjectStorageMetrics         []UserStatsMetrics
}
