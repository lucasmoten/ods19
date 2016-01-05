// This file is automatically generated. Do not modify.

package aac

import (
	"fmt"
)

var _ = fmt.Sprintf

type AcmInfo struct {
	Path            string `thrift:"1,required" json:"path"`
	Acm             string `thrift:"2,required" json:"acm"`
	IncludeInRollup bool   `thrift:"3,required" json:"includeInRollup"`
}

type AcmResponse struct {
	Success   bool     `thrift:"1,required" json:"success"`
	Messages  []string `thrift:"2,required" json:"messages"`
	AcmValid  bool     `thrift:"3,required" json:"acmValid"`
	HasAccess bool     `thrift:"4,required" json:"hasAccess"`
	AcmInfo   *AcmInfo `thrift:"5,required" json:"acmInfo"`
}

type AcmsForRollupWithPath struct {
	AcmInfo *AcmInfo `thrift:"1,required" json:"acmInfo"`
	Path    string   `thrift:"2,required" json:"path"`
}

type CheckAccessAndPopulateResponse struct {
	Success           bool           `thrift:"1,required" json:"success"`
	Messages          []string       `thrift:"2,required" json:"messages"`
	AcmResponseList   []*AcmResponse `thrift:"3,required" json:"AcmResponseList"`
	RollupAcmResponse *AcmResponse   `thrift:"4,required" json:"rollupAcmResponse"`
}

type CheckAccessResponse struct {
	Success   bool     `thrift:"1,required" json:"success"`
	Messages  []string `thrift:"2,required" json:"messages"`
	HasAccess bool     `thrift:"3,required" json:"hasAccess"`
}

type ClearUserAttributesResponse struct {
	Success  bool     `thrift:"1,required" json:"success"`
	Messages []string `thrift:"2,required" json:"messages"`
}

type RejectAccessResponse struct {
	Messages  []string `thrift:"1,required" json:"messages"`
	HasAccess bool     `thrift:"2,required" json:"hasAccess"`
}

type ShareResponse struct {
	Success  bool     `thrift:"1,required" json:"success"`
	Messages []string `thrift:"2,required" json:"messages"`
	Share    string   `thrift:"3,required" json:"share"`
}

type SimpleAcmResponse struct {
	Messages              []string `thrift:"1,required" json:"messages"`
	BodyWithValidatedAcms string   `thrift:"2,required" json:"bodyWithValidatedAcms"`
}

type SnippetResponse struct {
	Success  bool     `thrift:"1,required" json:"success"`
	Messages []string `thrift:"2,required" json:"messages"`
	Snippets string   `thrift:"3,required" json:"snippets"`
}

type UserAttributesResponse struct {
	Success        bool     `thrift:"1,required" json:"success"`
	Messages       []string `thrift:"2,required" json:"messages"`
	UserAttributes string   `thrift:"3,required" json:"userAttributes"`
}

type ValidateAcmsResponse struct {
	Success           bool           `thrift:"1,required" json:"success"`
	Messages          []string       `thrift:"2,required" json:"messages"`
	AcmResponseList   []*AcmResponse `thrift:"3,required" json:"AcmResponseList"`
	RollupAcmResponse *AcmResponse   `thrift:"4,required" json:"rollupAcmResponse"`
}

type ValidateTrigraphResponse struct {
	Success       bool `thrift:"1,required" json:"success"`
	TrigraphValid bool `thrift:"2,required" json:"trigraphValid"`
}

type InvalidInputException struct {
	Message string `thrift:"1,required" json:"message"`
}

func (e *InvalidInputException) Error() string {
	return fmt.Sprintf("InvalidInputException{Message: %+v}", e.Message)
}

type SecurityServiceException struct {
	Message string `thrift:"1,required" json:"message"`
}

func (e *SecurityServiceException) Error() string {
	return fmt.Sprintf("SecurityServiceException{Message: %+v}", e.Message)
}

type AacService interface {
	BuildAcm(byteList []byte, dataType string, propertiesMap map[string]string) (*AcmResponse, error)
	CheckAccess(userToken string, tokenType string, acm string) (*CheckAccessResponse, error)
	CheckAccessAndPopulate(userToken string, tokenType string, acmInfoList []*AcmInfo, calculateRollup bool, shareType string, share string) (*CheckAccessAndPopulateResponse, error)
	ClearUserAttributesFromCache(userToken string, tokenType string) (*ClearUserAttributesResponse, error)
	CreateAcmFromBannerMarking(banner string, shareType string, share string) (*AcmResponse, error)
	GetShare(userToken string, tokenType string, shareType string, share string) (*ShareResponse, error)
	GetSnippets(userToken string, tokenType string, snippetType string) (*SnippetResponse, error)
	GetUserAttributes(userToken string, tokenType string, snippetType string) (*UserAttributesResponse, error)
	IsCountryTrigraph(trigraph string) (*ValidateTrigraphResponse, error)
	PopulateAndValidateAcm(acm string) (*AcmResponse, error)
	PopulateAndValidateAcmFromCapcoString(capcoString string, capcoStringTypes string) (*AcmResponse, error)
	RollupAcms(userToken string, acmList []string, shareType string, share string) (*AcmResponse, error)
	ValidateAcm(acm string) (*AcmResponse, error)
	ValidateAcms(acmInfoList []*AcmInfo, userToken string, tokenType string, shareType string, share string, rollup bool, populate bool) (*ValidateAcmsResponse, error)
}

type AacServiceServer struct {
	Implementation AacService
}

func (s *AacServiceServer) BuildAcm(req *AacServiceBuildAcmRequest, res *AacServiceBuildAcmResponse) error {
	val, err := s.Implementation.BuildAcm(req.ByteList, req.DataType, req.PropertiesMap)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) CheckAccess(req *AacServiceCheckAccessRequest, res *AacServiceCheckAccessResponse) error {
	val, err := s.Implementation.CheckAccess(req.UserToken, req.TokenType, req.Acm)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) CheckAccessAndPopulate(req *AacServiceCheckAccessAndPopulateRequest, res *AacServiceCheckAccessAndPopulateResponse) error {
	val, err := s.Implementation.CheckAccessAndPopulate(req.UserToken, req.TokenType, req.AcmInfoList, req.CalculateRollup, req.ShareType, req.Share)
	switch e := err.(type) {
	case *SecurityServiceException:
		res.Ex1 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) ClearUserAttributesFromCache(req *AacServiceClearUserAttributesFromCacheRequest, res *AacServiceClearUserAttributesFromCacheResponse) error {
	val, err := s.Implementation.ClearUserAttributesFromCache(req.UserToken, req.TokenType)
	switch e := err.(type) {
	case *SecurityServiceException:
		res.Ex1 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) CreateAcmFromBannerMarking(req *AacServiceCreateAcmFromBannerMarkingRequest, res *AacServiceCreateAcmFromBannerMarkingResponse) error {
	val, err := s.Implementation.CreateAcmFromBannerMarking(req.Banner, req.ShareType, req.Share)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) GetShare(req *AacServiceGetShareRequest, res *AacServiceGetShareResponse) error {
	val, err := s.Implementation.GetShare(req.UserToken, req.TokenType, req.ShareType, req.Share)
	switch e := err.(type) {
	case *SecurityServiceException:
		res.Ex1 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) GetSnippets(req *AacServiceGetSnippetsRequest, res *AacServiceGetSnippetsResponse) error {
	val, err := s.Implementation.GetSnippets(req.UserToken, req.TokenType, req.SnippetType)
	switch e := err.(type) {
	case *SecurityServiceException:
		res.Ex1 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) GetUserAttributes(req *AacServiceGetUserAttributesRequest, res *AacServiceGetUserAttributesResponse) error {
	val, err := s.Implementation.GetUserAttributes(req.UserToken, req.TokenType, req.SnippetType)
	switch e := err.(type) {
	case *SecurityServiceException:
		res.Ex1 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) IsCountryTrigraph(req *AacServiceIsCountryTrigraphRequest, res *AacServiceIsCountryTrigraphResponse) error {
	val, err := s.Implementation.IsCountryTrigraph(req.Trigraph)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) PopulateAndValidateAcm(req *AacServicePopulateAndValidateAcmRequest, res *AacServicePopulateAndValidateAcmResponse) error {
	val, err := s.Implementation.PopulateAndValidateAcm(req.Acm)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) PopulateAndValidateAcmFromCapcoString(req *AacServicePopulateAndValidateAcmFromCapcoStringRequest, res *AacServicePopulateAndValidateAcmFromCapcoStringResponse) error {
	val, err := s.Implementation.PopulateAndValidateAcmFromCapcoString(req.CapcoString, req.CapcoStringTypes)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) RollupAcms(req *AacServiceRollupAcmsRequest, res *AacServiceRollupAcmsResponse) error {
	val, err := s.Implementation.RollupAcms(req.UserToken, req.AcmList, req.ShareType, req.Share)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) ValidateAcm(req *AacServiceValidateAcmRequest, res *AacServiceValidateAcmResponse) error {
	val, err := s.Implementation.ValidateAcm(req.Acm)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

func (s *AacServiceServer) ValidateAcms(req *AacServiceValidateAcmsRequest, res *AacServiceValidateAcmsResponse) error {
	val, err := s.Implementation.ValidateAcms(req.AcmInfoList, req.UserToken, req.TokenType, req.ShareType, req.Share, req.Rollup, req.Populate)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *SecurityServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

type AacServiceBuildAcmRequest struct {
	ByteList      []byte            `thrift:"1,required" json:"byteList"`
	DataType      string            `thrift:"2,required" json:"dataType"`
	PropertiesMap map[string]string `thrift:"3,required" json:"propertiesMap"`
}

type AacServiceBuildAcmResponse struct {
	Value *AcmResponse              `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServiceCheckAccessRequest struct {
	UserToken string `thrift:"1,required" json:"userToken"`
	TokenType string `thrift:"2,required" json:"tokenType"`
	Acm       string `thrift:"3,required" json:"acm"`
}

type AacServiceCheckAccessResponse struct {
	Value *CheckAccessResponse      `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServiceCheckAccessAndPopulateRequest struct {
	UserToken       string     `thrift:"1,required" json:"userToken"`
	TokenType       string     `thrift:"2,required" json:"tokenType"`
	AcmInfoList     []*AcmInfo `thrift:"3,required" json:"acmInfoList"`
	CalculateRollup bool       `thrift:"4,required" json:"calculateRollup"`
	ShareType       string     `thrift:"5,required" json:"shareType"`
	Share           string     `thrift:"6,required" json:"share"`
}

type AacServiceCheckAccessAndPopulateResponse struct {
	Value *CheckAccessAndPopulateResponse `thrift:"0" json:"value,omitempty"`
	Ex1   *SecurityServiceException       `thrift:"1" json:"ex1,omitempty"`
}

type AacServiceClearUserAttributesFromCacheRequest struct {
	UserToken string `thrift:"1,required" json:"userToken"`
	TokenType string `thrift:"2,required" json:"tokenType"`
}

type AacServiceClearUserAttributesFromCacheResponse struct {
	Value *ClearUserAttributesResponse `thrift:"0" json:"value,omitempty"`
	Ex1   *SecurityServiceException    `thrift:"1" json:"ex1,omitempty"`
}

type AacServiceCreateAcmFromBannerMarkingRequest struct {
	Banner    string `thrift:"1,required" json:"banner"`
	ShareType string `thrift:"2,required" json:"shareType"`
	Share     string `thrift:"3,required" json:"share"`
}

type AacServiceCreateAcmFromBannerMarkingResponse struct {
	Value *AcmResponse              `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServiceGetShareRequest struct {
	UserToken string `thrift:"1,required" json:"userToken"`
	TokenType string `thrift:"2,required" json:"tokenType"`
	ShareType string `thrift:"3,required" json:"shareType"`
	Share     string `thrift:"4,required" json:"share"`
}

type AacServiceGetShareResponse struct {
	Value *ShareResponse            `thrift:"0" json:"value,omitempty"`
	Ex1   *SecurityServiceException `thrift:"1" json:"ex1,omitempty"`
}

type AacServiceGetSnippetsRequest struct {
	UserToken   string `thrift:"1,required" json:"userToken"`
	TokenType   string `thrift:"2,required" json:"tokenType"`
	SnippetType string `thrift:"3,required" json:"snippetType"`
}

type AacServiceGetSnippetsResponse struct {
	Value *SnippetResponse          `thrift:"0" json:"value,omitempty"`
	Ex1   *SecurityServiceException `thrift:"1" json:"ex1,omitempty"`
}

type AacServiceGetUserAttributesRequest struct {
	UserToken   string `thrift:"1,required" json:"userToken"`
	TokenType   string `thrift:"2,required" json:"tokenType"`
	SnippetType string `thrift:"3,required" json:"snippetType"`
}

type AacServiceGetUserAttributesResponse struct {
	Value *UserAttributesResponse   `thrift:"0" json:"value,omitempty"`
	Ex1   *SecurityServiceException `thrift:"1" json:"ex1,omitempty"`
}

type AacServiceIsCountryTrigraphRequest struct {
	Trigraph string `thrift:"1,required" json:"trigraph"`
}

type AacServiceIsCountryTrigraphResponse struct {
	Value *ValidateTrigraphResponse `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServicePopulateAndValidateAcmRequest struct {
	Acm string `thrift:"1,required" json:"acm"`
}

type AacServicePopulateAndValidateAcmResponse struct {
	Value *AcmResponse              `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServicePopulateAndValidateAcmFromCapcoStringRequest struct {
	CapcoString      string `thrift:"1,required" json:"capcoString"`
	CapcoStringTypes string `thrift:"2,required" json:"capcoStringTypes"`
}

type AacServicePopulateAndValidateAcmFromCapcoStringResponse struct {
	Value *AcmResponse              `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServiceRollupAcmsRequest struct {
	UserToken string   `thrift:"1,required" json:"userToken"`
	AcmList   []string `thrift:"2,required" json:"acmList"`
	ShareType string   `thrift:"3,required" json:"shareType"`
	Share     string   `thrift:"4,required" json:"share"`
}

type AacServiceRollupAcmsResponse struct {
	Value *AcmResponse              `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServiceValidateAcmRequest struct {
	Acm string `thrift:"1,required" json:"acm"`
}

type AacServiceValidateAcmResponse struct {
	Value *AcmResponse              `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServiceValidateAcmsRequest struct {
	AcmInfoList []*AcmInfo `thrift:"1,required" json:"acmInfoList"`
	UserToken   string     `thrift:"2,required" json:"userToken"`
	TokenType   string     `thrift:"3,required" json:"tokenType"`
	ShareType   string     `thrift:"4,required" json:"shareType"`
	Share       string     `thrift:"5,required" json:"share"`
	Rollup      bool       `thrift:"6,required" json:"rollup"`
	Populate    bool       `thrift:"7,required" json:"populate"`
}

type AacServiceValidateAcmsResponse struct {
	Value *ValidateAcmsResponse     `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException    `thrift:"1" json:"ex1,omitempty"`
	Ex2   *SecurityServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AacServiceClient struct {
	Client RPCClient
}

func (s *AacServiceClient) BuildAcm(byteList []byte, dataType string, propertiesMap map[string]string) (ret *AcmResponse, err error) {
	req := &AacServiceBuildAcmRequest{
		ByteList:      byteList,
		DataType:      dataType,
		PropertiesMap: propertiesMap,
	}
	res := &AacServiceBuildAcmResponse{}
	err = s.Client.Call("buildAcm", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) CheckAccess(userToken string, tokenType string, acm string) (ret *CheckAccessResponse, err error) {
	req := &AacServiceCheckAccessRequest{
		UserToken: userToken,
		TokenType: tokenType,
		Acm:       acm,
	}
	res := &AacServiceCheckAccessResponse{}
	err = s.Client.Call("checkAccess", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) CheckAccessAndPopulate(userToken string, tokenType string, acmInfoList []*AcmInfo, calculateRollup bool, shareType string, share string) (ret *CheckAccessAndPopulateResponse, err error) {
	req := &AacServiceCheckAccessAndPopulateRequest{
		UserToken:       userToken,
		TokenType:       tokenType,
		AcmInfoList:     acmInfoList,
		CalculateRollup: calculateRollup,
		ShareType:       shareType,
		Share:           share,
	}
	res := &AacServiceCheckAccessAndPopulateResponse{}
	err = s.Client.Call("checkAccessAndPopulate", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) ClearUserAttributesFromCache(userToken string, tokenType string) (ret *ClearUserAttributesResponse, err error) {
	req := &AacServiceClearUserAttributesFromCacheRequest{
		UserToken: userToken,
		TokenType: tokenType,
	}
	res := &AacServiceClearUserAttributesFromCacheResponse{}
	err = s.Client.Call("clearUserAttributesFromCache", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) CreateAcmFromBannerMarking(banner string, shareType string, share string) (ret *AcmResponse, err error) {
	req := &AacServiceCreateAcmFromBannerMarkingRequest{
		Banner:    banner,
		ShareType: shareType,
		Share:     share,
	}
	res := &AacServiceCreateAcmFromBannerMarkingResponse{}
	err = s.Client.Call("createAcmFromBannerMarking", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) GetShare(userToken string, tokenType string, shareType string, share string) (ret *ShareResponse, err error) {
	req := &AacServiceGetShareRequest{
		UserToken: userToken,
		TokenType: tokenType,
		ShareType: shareType,
		Share:     share,
	}
	res := &AacServiceGetShareResponse{}
	err = s.Client.Call("getShare", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) GetSnippets(userToken string, tokenType string, snippetType string) (ret *SnippetResponse, err error) {
	req := &AacServiceGetSnippetsRequest{
		UserToken:   userToken,
		TokenType:   tokenType,
		SnippetType: snippetType,
	}
	res := &AacServiceGetSnippetsResponse{}
	err = s.Client.Call("getSnippets", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) GetUserAttributes(userToken string, tokenType string, snippetType string) (ret *UserAttributesResponse, err error) {
	req := &AacServiceGetUserAttributesRequest{
		UserToken:   userToken,
		TokenType:   tokenType,
		SnippetType: snippetType,
	}
	res := &AacServiceGetUserAttributesResponse{}
	err = s.Client.Call("getUserAttributes", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) IsCountryTrigraph(trigraph string) (ret *ValidateTrigraphResponse, err error) {
	req := &AacServiceIsCountryTrigraphRequest{
		Trigraph: trigraph,
	}
	res := &AacServiceIsCountryTrigraphResponse{}
	err = s.Client.Call("isCountryTrigraph", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) PopulateAndValidateAcm(acm string) (ret *AcmResponse, err error) {
	req := &AacServicePopulateAndValidateAcmRequest{
		Acm: acm,
	}
	res := &AacServicePopulateAndValidateAcmResponse{}
	err = s.Client.Call("populateAndValidateAcm", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) PopulateAndValidateAcmFromCapcoString(capcoString string, capcoStringTypes string) (ret *AcmResponse, err error) {
	req := &AacServicePopulateAndValidateAcmFromCapcoStringRequest{
		CapcoString:      capcoString,
		CapcoStringTypes: capcoStringTypes,
	}
	res := &AacServicePopulateAndValidateAcmFromCapcoStringResponse{}
	err = s.Client.Call("populateAndValidateAcmFromCapcoString", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) RollupAcms(userToken string, acmList []string, shareType string, share string) (ret *AcmResponse, err error) {
	req := &AacServiceRollupAcmsRequest{
		UserToken: userToken,
		AcmList:   acmList,
		ShareType: shareType,
		Share:     share,
	}
	res := &AacServiceRollupAcmsResponse{}
	err = s.Client.Call("rollupAcms", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) ValidateAcm(acm string) (ret *AcmResponse, err error) {
	req := &AacServiceValidateAcmRequest{
		Acm: acm,
	}
	res := &AacServiceValidateAcmResponse{}
	err = s.Client.Call("validateAcm", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}

func (s *AacServiceClient) ValidateAcms(acmInfoList []*AcmInfo, userToken string, tokenType string, shareType string, share string, rollup bool, populate bool) (ret *ValidateAcmsResponse, err error) {
	req := &AacServiceValidateAcmsRequest{
		AcmInfoList: acmInfoList,
		UserToken:   userToken,
		TokenType:   tokenType,
		ShareType:   shareType,
		Share:       share,
		Rollup:      rollup,
		Populate:    populate,
	}
	res := &AacServiceValidateAcmsResponse{}
	err = s.Client.Call("validateAcms", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}
