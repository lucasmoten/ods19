package protocol

// FilterSetting denotes a field and a condition to match an expression on which to filter results
type FilterSetting struct {
	// FilterField represents the field that results are being filtered on
	FilterField string `json:"filterField"`
	// Condition is the match type for filtering (e.g. begins, contains, ends, equals)
	Condition string `json:"condition"`
	// Expression is a phrase used in relation to the FilterField by condition
	Expression string `json:"expression"`
}
