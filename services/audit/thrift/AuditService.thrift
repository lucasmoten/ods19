namespace * gov.ic.dodiis.dctc.bedrock.audit.thrift

include "events.thrift"

typedef i32 int

/**
 *  Audit response status object.
 *
 *  @param success. True if audit request completed successfully, otherwise set to false
 *  @param messages. List of any error or warning messages that may have occurred
 */
struct AuditResponse {
    1: bool success,
    2: list<string> messages
}

/** Invalid input */
exception InvalidInputException{
    1: string message
}

/** catch all exception */
exception AuditServiceException{
    1: string message
}

service AuditService {

  string ping(),
  
  /**
    * Submits an Audit Record
	*
	* @param event. Thirft representation of the Audit.XML event
	*
	* @throws InvalidInputException
	* @throws AuditServiceException
  */
  AuditResponse submitAuditEvent(1: events.AuditEvent event) throws(1: InvalidInputException ex1, 2: AuditServiceException ex2)

}