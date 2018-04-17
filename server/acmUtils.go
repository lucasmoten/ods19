package server

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/net/context"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/utils"
)

func createMapFromInterface(sourceInterface interface{}) (map[string]interface{}, bool) {
	m, ok := sourceInterface.(map[string]interface{})
	return m, ok
}
func getACMMap(obj *models.ODObject) (*AppError, map[string]interface{}) {
	if !obj.RawAcm.Valid {
		return NewAppError(http.StatusBadRequest, fmt.Errorf("The object has no valid ACM"), "Missing ACM"), nil
	}
	if len(obj.RawAcm.String) == 0 {
		return NewAppError(http.StatusBadRequest, fmt.Errorf("The object has no ACM"), "Missing ACM"), nil
	}
	acmInterface, err := utils.UnmarshalStringToInterface(obj.RawAcm.String)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "ACM unparseable"), nil
	}
	acmMap, ok := createMapFromInterface(acmInterface)
	if !ok {
		return NewAppError(http.StatusInternalServerError, fmt.Errorf("ACM does not convert to map"), "ACM unparseable"), nil
	}
	return nil, acmMap
}
func getACMInterfacePart(obj *models.ODObject, acmKeySearch string) (*AppError, interface{}) {
	herr, acmMap := getACMMap(obj)
	if herr != nil {
		return herr, nil
	}
	var foundInterface interface{}
	for acmKey, acmValue := range acmMap {
		if strings.Compare(acmKey, acmKeySearch) == 0 {
			foundInterface = acmValue
			break
		}
	}
	return nil, foundInterface
}

func removeACMPart(ctx context.Context, obj *models.ODObject, acmKeySearch string) *AppError {
	return setACMPartFromInterface(ctx, obj, acmKeySearch, nil)
}
func setACMPartFromStringArray(ctx context.Context, obj *models.ODObject, acmKeySearch string, acmValues []string) *AppError {
	// Build the interface array of the values
	interfaceArray := make([]interface{}, len(acmValues))
	for i, v := range acmValues {
		interfaceArray[i] = v
	}
	// Assign it
	return setACMPartFromInterfaceArray(ctx, obj, acmKeySearch, interfaceArray)
}

func setACMPartFromInterfaceArray(ctx context.Context, obj *models.ODObject, acmKeySearch string, interfaceArray []interface{}) *AppError {
	return setACMPartFromInterface(ctx, obj, acmKeySearch, interfaceArray)
}
func setACMPartFromInterface(ctx context.Context, obj *models.ODObject, acmKeySearch string, interfaceValue interface{}) *AppError {
	// Get the map
	herr, acmMap := getACMMap(obj)
	if herr != nil {
		return herr
	}
	// Assign to the key in the map
	if interfaceValue != nil {
		acmMap[acmKeySearch] = interfaceValue
	} else {
		delete(acmMap, acmKeySearch)
	}
	// Convert to string
	newACM, err := utils.MarshalInterfaceToString(acmMap)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "unable to update acm")
	}
	normalizedNewACM, err := utils.NormalizeMarshalledInterface(newACM)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "unable to normalize new acm")
	}
	normalizedOriginalACM, err := utils.NormalizeMarshalledInterface(obj.RawAcm.String)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "unable to normalize original acm")
	}
	if strings.Compare(normalizedNewACM, normalizedOriginalACM) != 0 {
		LoggerFromContext(ctx).Debug("changing vlaue of acm", zap.String("original acm", obj.RawAcm.String), zap.String("normalized original acm", normalizedOriginalACM), zap.String("normalized new acm", normalizedNewACM))
		obj.RawAcm.String = normalizedNewACM
	}
	return nil
}

func getStringArrayFromInterface(i interface{}) []string {
	var o []string
	if i != nil {
		v := i.([]interface{})
		for _, e := range v {
			if strings.Compare(reflect.TypeOf(e).Kind().String(), "string") == 0 {
				o = append(o, e.(string))
			}
		}
	}
	return o
}

func isAcmShareDifferent(acm1 string, acm2 string) (bool, *AppError) {

	different := true

	obj1 := models.ODObject{RawAcm: models.ToNullString(acm1)}
	obj2 := models.ODObject{RawAcm: models.ToNullString(acm2)}

	herr1, acmShareInterface1 := getACMInterfacePart(&obj1, "share")
	if herr1 != nil {
		return different, herr1
	}
	herr2, acmShareInterface2 := getACMInterfacePart(&obj2, "share")
	if herr2 != nil {
		return different, herr2
	}

	same := reflect.DeepEqual(acmShareInterface1, acmShareInterface2)
	return !same, nil
}
