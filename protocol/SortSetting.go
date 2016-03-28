package protocol

// SortSetting denotes a field and a preferred direction on which to sort results.
type SortSetting struct {
	SortField     string `json:"sortField"`
	SortAscending bool   `json:"sortAscending"`
}
