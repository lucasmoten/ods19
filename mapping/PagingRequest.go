package mapping

import (
	"github.com/deciphernow/object-drive-server/dao"
	"github.com/deciphernow/object-drive-server/protocol"
)

// MapPagingRequestToDAOPagingRequest converts a protocol PagingRequest to the
// similarl structured PagingRequest in the dao package for use in database calls
func MapPagingRequestToDAOPagingRequest(i *protocol.PagingRequest) dao.PagingRequest {
	o := dao.PagingRequest{PageNumber: i.PageNumber, PageSize: i.PageSize}
	o.FilterSettings = mapFilterSettingsToDAOFilterSettings(&i.FilterSettings)
	o.SortSettings = mapSortSettingsToDAOSortSettings(&i.SortSettings)
	o.FilterMatchType = i.FilterMatchType
	return o
}

func mapFilterSettingsToDAOFilterSettings(i *[]protocol.FilterSetting) []dao.FilterSetting {
	o := make([]dao.FilterSetting, len(*i))
	for p, q := range *i {
		o[p] = mapFilterSettingToDAOFilterSetting(&q)
	}
	return o
}

func mapFilterSettingToDAOFilterSetting(i *protocol.FilterSetting) dao.FilterSetting {
	return dao.FilterSetting{FilterField: i.FilterField, Condition: i.Condition, Expression: i.Expression}
}

func mapSortSettingsToDAOSortSettings(i *[]protocol.SortSetting) []dao.SortSetting {
	o := make([]dao.SortSetting, len(*i))
	for p, q := range *i {
		o[p] = mapSortSettingToDAOSortSetting(&q)
	}
	return o
}

func mapSortSettingToDAOSortSetting(i *protocol.SortSetting) dao.SortSetting {
	return dao.SortSetting{SortField: i.SortField, SortAscending: i.SortAscending}
}
