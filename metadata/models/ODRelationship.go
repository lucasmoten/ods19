package models

/*
ODRelationship is a structure defining the associative attributes linking
two Objects to each other within Object Drive.
*/
type ODRelationship struct {
	ODCommonMeta
	ODChangeTracking
	SourceID    []byte     `db:"sourceId"`
	TargetID    []byte     `db:"targetId"`
	Description NullString `db:"description"`
	/*
		ClassificationPM is the portion mark classification for the description of
		this relationship
	*/
	ClassificationPM NullString `db:"classificationPM"`
}
