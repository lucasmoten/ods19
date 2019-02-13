package protocol

// SortSetting denotes a field and a preferred direction on which to sort results.
type SortSetting struct {
	// SortField indicates the field name for which results should be sorted
	SortField string `json:"sortField"`
	// SortAscending denotes whether to sort by the field in ascending or descending order
	SortAscending bool `json:"sortAscending"`
}
