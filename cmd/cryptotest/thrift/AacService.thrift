namespace java gov.ic.cte.acm.service
namespace go aac

/**
 *  ACM object optionally identified by a path
 *
 *  @param path. Path of the ACM in the source system. This is optional
 *  @param acm. JSON representation of ACM
 *  @param includeInRollup. If true, this ACM will be included in the rollup
 */
struct AcmInfo{
    1: string path,
    2: string acm,
    3: bool includeInRollup
}

/**
 *  Response for acm rollup with path
 *
 *  @param acmInfo. 
 *  @param path. 
 */
struct AcmsForRollupWithPath{
	1: AcmInfo acmInfo,
	2: string path
}

/**
 *  ACM status object. acmString is populated by the methods which return an ACM on 
 *  success
 *
 *  @param success. True if request was successfully processed, otherwise false
 *  @param messages. List of error/warning messages
 *  @param acmValid. True if ACM in the request is valid
 *  @param hasAccess. True, if the user has access to the ACM
 *  @param AcmInfo. ACM info
 */
struct AcmResponse {
    1: bool success,
    2: list<string> messages,
    3: bool acmValid,
    4: bool hasAccess,
    5: AcmInfo acmInfo
}

/**
 *  Check access status object.
 *
 *  @param success. True if request was successfully processed, otherwise false
 *  @param messages. List of error/warning messages
 *  @param hasAccess. True if user attributes have access to ACM
 */
struct CheckAccessResponse {
    1: bool success,
    2: list<string> messages,
    3: bool hasAccess
}

/**
 *  Reject access response object.
 *
 *  @param messages. List of error/warning messages
 *  @param hasAccess. True if user attributes have access to ACM
 */
struct RejectAccessResponse {
    1: list<string> messages,
    2: bool hasAccess
}

/**
 *  Retrieve/populate user attributes response object
 *
 *  @param success. True if request was successfully processed, otherwise false
 *  @param messages. List of error/warning messages
 *  @param userAttributesString. JSON representation of output user attributes
 */
struct UserAttributesResponse{
    1: bool success,
    2: list<string> messages,
    3: string userAttributes
}

/**
 *  Clear user attributes response object
 *
 *  @param success. True if request was successfully processed, otherwise false
 *  @param messages. List of error/warning messages
 */
struct ClearUserAttributesResponse{
    1: bool success,
    2: list<string> messages
}

/**
 *  Get snippet response object
 *
 *  @param success. True if request was successfully processed, otherwise false
 *  @param messages. List of error/warning messages
 *  @param snippets. Snippets in JSON form keyed by the snippet name.
 */
struct SnippetResponse{
    1: bool success,
    2: list<string> messages,
    3: string snippets
}

/**
 *  Get share response object
 *
 *  @param success. True if request was successfully processed, otherwise false
 *  @param messages. List of error/warning messages
 *  @param share. JSON representation of the share object
 */
struct ShareResponse{
    1: bool success,
    2: list<string> messages,
    3: string share
}

/**
 * Response object for checkAccessAndPopulate.
 *
 *  @param  AcmResponseList.  List of ACM status objects
 *  @param  acmRollup.  Rollup of all ACM objects
 */
struct CheckAccessAndPopulateResponse{
    1: bool success,
    2: list<string> messages,
    3: list<AcmResponse> AcmResponseList,
    4: AcmResponse rollupAcmResponse
}

struct ValidateAcmsResponse{
    1: bool success,
    2: list<string> messages,
    3: list<AcmResponse> AcmResponseList,
    4: AcmResponse rollupAcmResponse
}

struct SimpleAcmResponse{
	1: list<string> messages,
	2: string bodyWithValidatedAcms
}

struct ValidateTrigraphResponse{
    1: bool success,
    2: bool trigraphValid
}

/** Invalid input */
exception InvalidInputException{
    1: string message
}

/** catch all exception */
exception SecurityServiceException{
    1: string message
}


service AacService {
	/** 
	 *	Builds an ACM object from a byte string. Used by message traffic ingestion
	 *	
	 *	@param byteList. Data to be converted to ACM
	 *	@param dataTye. Data type. Supported types: XML
	 *	@param propertiesMap. Optional properties used to build ACM
	 *
	 *	@throws InvalidInputException
	 *	@throws SecurityServiceException
	 */
	AcmResponse buildAcm(1: list<byte> byteList, 2: string dataType,
					   3: map<string, string> propertiesMap)
		throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),

	/** 
	 *	Validates ACM list, will optionally do rollup and populate.
	 *	
	 *	@param acm. JSON representation of the ACM
	 *
	 *	@throws InvalidInputException
	 *	@throws SecurityServiceException
	 */
	ValidateAcmsResponse validateAcms(1: list<AcmInfo> acmInfoList, 2: string userToken, 3: string tokenType,
	                        4: string shareType, 5: string share,
	                        6: bool rollup, 7: bool populate)
		throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),

    /** 
     *  Validates ACM
     *  
     *  @param acm. JSON representation of the ACM
     *
     *  @throws InvalidInputException
     *  @throws SecurityServiceException
     */
    AcmResponse validateAcm(1: string acm)
        throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),
        
	/** 
	 *	Auto populates normalized ACM fields and validates ACM
	 *	
	 *	@param acm. JSON representation of the ACM
	 *
	 *	@throws InvalidInputException
	 *	@throws SecurityServiceException
	 */
	AcmResponse populateAndValidateAcm(1: string acm)
		throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),

    /**
     *	Creates an ACM object from a capco string, normalizes and validates the ACM
     *
     *	@param capcoString.  cacpo string
     *  @param capcoStringTypes.  corresponds to CapcoStringType enum ("TITLE","ABBREVIATION","PORTIONMARKING","ISM")
     *
     *	@throws InvalidInputException
     *	@throws SecurityServiceException
     */
    AcmResponse populateAndValidateAcmFromCapcoString(1: string capcoString, 2: string capcoStringTypes)
        throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),

    /**
     *	Detemines whether a country trigraph is valid
     *
     *	@param trigraph.  string representing the country trigraph to be checked
     *
     *	@throws InvalidInputException
     *	@throws SecurityServiceException
     */
     ValidateTrigraphResponse isCountryTrigraph(1: string trigraph)
        throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),

    /**
     *  Creates an ACM object from a banner marking.
     *
     *  @param banner. Banner marking
	 *  @param shareType. share type, valid values - public, private, other
	 *	@param share. Share object if share type is other
     *
     *  @throws InvalidInputException
     *	@throws SecurityServiceException
     */
    AcmResponse createAcmFromBannerMarking(1: string banner, 2: string shareType, 3: string share)
        throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),

	/** 
	 *	Rolls up ACMs and populates normalized values in rolled up ACM
	 *
	 *  @param userToken. User token, required if share type is public
	 *	@param acmList. List of ACMs in JSON format
	 *  @param shareType. share type, valid values - public, private, other
	 *	@param share. Share object if share type is other
	 *
	 *	@throws InvalidInputException
	 *	@throws SecurityServiceException
	 */
	AcmResponse rollupAcms(1: string userToken, 2: list<string> acmList, 3: string shareType,
	    4: string share) throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),

	/**
	 *	Checks if the user attributes have access to ACM
	 *
     *  @param userToken. User token
     *  @param tokenType. Type of user token
	 *	@param acm. JSON representation of the ACM
	 *
	 *	@throws InvalidInputException
	 *	@throws SecurityServiceException
	 */
	CheckAccessResponse checkAccess(1: string userToken, 2: string tokenType, 3: string acm)
		throws(1: InvalidInputException ex1, 2: SecurityServiceException ex2),

    /**
     *  For each ACM object in the list, checks to see if user has access to it. If user has access, it validates
     *  and populates ACM.
     *
     *  @param userToken. User token
     *  @param tokenType. Type of user token
     *  @param acmInfoList. List of ACMs optionally identified with paths
     *  @param calculateRollup. If true, a rollup of all ACMs will be performed and returned in the response
     *  @param shareType. Share type of the rolled up. Supported types are public, private, and other.
     *  @param share. Share object if share type is other.
     *
     *  @throws SecurityServiceException
     */
    CheckAccessAndPopulateResponse checkAccessAndPopulate(1: string userToken, 2: string tokenType,
         3: list<AcmInfo> acmInfoList, 4: bool calculateRollup, 5: string shareType, 6: string share)
         throws(1: SecurityServiceException ex1)

	/**
	 *  Retrieves a user's attributes
	 *
	 *  @param userToken. User token
	 *  @param tokenType. Type of user token
	 *  @param snippetType. If specified, snippets of the specified type are returned. Supported types are Mongo and ES
	 *
	 *  @throws SecurityServiceException
	 */
	UserAttributesResponse getUserAttributes(1: string userToken, 2: string tokenType, 3: string snippetType)
	    throws(1: SecurityServiceException ex1),

    /**
     * Clears user attributes from cache
     *
     *  @param userToken. User token
     *  @param tokenType. Type of user token
     *
     *  @throws SecurityServiceException
     */
    ClearUserAttributesResponse clearUserAttributesFromCache(1: string userToken, 2: string tokenType)
        throws(1: SecurityServiceException ex1),

    /**
     *  Generates snippets of a given type based on the user's attributes
     *
     *  @param userToken. User token
     *  @param tokenType. Type of user token
     *  @param type. Snippet type. Supported types are Mongo and ES.
     *
     *  @throws SecurityServiceException
     */
	SnippetResponse getSnippets(1: string userToken, 2: string tokenType, 3: string snippetType)
	    throws (1: SecurityServiceException ex1)

    /**
     * Generates a share object for a given type.
     *
     *  @param userToken. User token
     *  @param tokenType. Type of user token
     *  @param shareType. Share type. Supported types are public, private, and other.
     *  @param share. Share object if share type is other.
     *
     *  @throws SecurityServiceException
     */
    ShareResponse getShare(1: string userToken, 2: string tokenType, 3: string shareType, 4: string share)
        throws (1: SecurityServiceException ex1)
}
