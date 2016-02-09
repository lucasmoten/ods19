package models

// ODRelationship is a structure defining the associative attributes linking
// two Objects to each other within Object Drive.
type ODRelationship struct {
	ODCommonMeta
	ODChangeTracking
	// SourceID is the identifier of the source object for this relationship.
	// In a hierarchial context, the source refers to the 'parent' item
	SourceID []byte `db:"sourceId"`
	// TargetID is the identifier of the target object for this relationship.
	// In a hierarchial context, the target refers to the 'child' item
	TargetID []byte `db:"targetId"`
	// Description is an indicator of the purpose of the relationship.
	Description NullString `db:"description"`
	// ClassificationPM is the portion mark classification for the description of
	// this relationship
	ClassificationPM NullString `db:"classificationPM"`
}
