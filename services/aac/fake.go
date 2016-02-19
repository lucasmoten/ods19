package aac

// FakeAAC holds AAC service response objects for mocking in unit tests.
type FakeAAC struct {
	ACMResp                    *AcmResponse
	CheckAccessResp            *CheckAccessResponse
	CheckAccessAndPopulateResp *CheckAccessAndPopulateResponse
	ClearUserAttributesResp    *ClearUserAttributesResponse
	ShareResp                  *ShareResponse
	SnippetResp                *SnippetResponse
	Err                        error
	UserAttributesResp         *UserAttributesResponse
	ValidateTrigraphResp       *ValidateTrigraphResponse
	ValidateAcmsResp           *ValidateAcmsResponse
}

// BuildAcm for FakeAAC.
func (fake *FakeAAC) BuildAcm(
	byteList []int8, dataType string, propertiesMap map[string]string) (*AcmResponse, error) {
	return fake.ACMResp, fake.Err
}

// CheckAccess for FakeAAC.
func (fake *FakeAAC) CheckAccess(userToken string, tokenType string, acm string) (*CheckAccessResponse, error) {
	return fake.CheckAccessResp, fake.Err
}

// CheckAccessAndPopulate for FakeAAC.
func (fake *FakeAAC) CheckAccessAndPopulate(
	userToken string, tokenType string, acmInfoList []*AcmInfo, calculateRollup bool, shareType string, share string) (*CheckAccessAndPopulateResponse, error) {
	return fake.CheckAccessAndPopulateResp, fake.Err
}

// ClearUserAttributesFromCache for FakeAAC.
func (fake *FakeAAC) ClearUserAttributesFromCache(userToken string, tokenType string) (*ClearUserAttributesResponse, error) {
	return fake.ClearUserAttributesResp, fake.Err
}

// CreateAcmFromBannerMarking for FakeAAC.
func (fake *FakeAAC) CreateAcmFromBannerMarking(banner string, shareType string, share string) (*AcmResponse, error) {
	return fake.ACMResp, fake.Err
}

// GetShare for FakeAAC.
func (fake *FakeAAC) GetShare(userToken string, tokenType string, shareType string, share string) (*ShareResponse, error) {
	return fake.ShareResp, fake.Err
}

// GetSnippets for FakeAAC.
func (fake *FakeAAC) GetSnippets(userToken string, tokenType string, snippetType string) (*SnippetResponse, error) {
	return fake.SnippetResp, fake.Err
}

// GetUserAttributes for FakeAAC.
func (fake *FakeAAC) GetUserAttributes(userToken string, tokenType string, snippetType string) (*UserAttributesResponse, error) {
	return fake.UserAttributesResp, fake.Err
}

// IsCountryTrigraph for FakeAAC.
func (fake *FakeAAC) IsCountryTrigraph(trigraph string) (*ValidateTrigraphResponse, error) {
	return fake.ValidateTrigraphResp, fake.Err
}

// PopulateAndValidateAcm for FakeAAC.
func (fake *FakeAAC) PopulateAndValidateAcm(acm string) (*AcmResponse, error) {
	return fake.ACMResp, fake.Err
}

// PopulateAndValidateAcmFromCapcoString for FakeAAC.
func (fake *FakeAAC) PopulateAndValidateAcmFromCapcoString(capcoString string, capcoStringTypes string) (*AcmResponse, error) {
	return fake.ACMResp, fake.Err
}

// RollupAcms for FakeAAC.
func (fake *FakeAAC) RollupAcms(userToken string, acmList []string, shareType string, share string) (*AcmResponse, error) {
	return fake.ACMResp, fake.Err
}

// ValidateAcm for FakeAAC.
func (fake *FakeAAC) ValidateAcm(acm string) (*AcmResponse, error) {
	return fake.ACMResp, fake.Err
}

// ValidateAcms for FakeAAC.
func (fake *FakeAAC) ValidateAcms(acmInfoList []*AcmInfo, userToken string, tokenType string, shareType string, share string, rollup bool, populate bool) (*ValidateAcmsResponse, error) {
	return fake.ValidateAcmsResp, fake.Err
}
