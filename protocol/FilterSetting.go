package protocol

// FilterSetting denotes a field and a condition to match an expression on which to filter results
type FilterSetting struct {
	FilterField string `json:"filterField"`
	Condition   string `json:"condition"`
	Expression  string `json:"expression"`
}
