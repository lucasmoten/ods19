package dao

// GetSanitizedPageNumber takes an input number, and ensures that it is no less
// than 1
func GetSanitizedPageNumber(pageNumber int) int {
	if pageNumber < 1 {
		return 1
	}
	return pageNumber
}

// GetSanitizedPageSize takes an input number, and ensures it is within the
// range of 1 .. 10000
func GetSanitizedPageSize(pageSize int) int {
	if pageSize < 1 {
		return 1
	}
	if pageSize > 10000 {
		return 10000
	}
	return pageSize
}

// GetLimit is used for determining the upper bound of records to request from
// the database, specifically pageNumber * pageSize
func GetLimit(pageNumber int, pageSize int) int {
	return GetSanitizedPageNumber(pageNumber) * GetSanitizedPageSize(pageSize)
}

// GetOffset is used for determining the lower bound of records to request from
// the database, starting with the first item on a given page based on size
func GetOffset(pageNumber int, pageSize int) int {
	return GetLimit(pageNumber, pageSize) - pageSize
}

// GetPageCount determines the total number of pages that would exist when the
// totalRows and pageSize are known
func GetPageCount(totalRows int, pageSize int) int {
	pageCount := totalRows / pageSize
	for (pageCount * pageSize) < totalRows {
		pageCount++
	}
	return pageCount
}
