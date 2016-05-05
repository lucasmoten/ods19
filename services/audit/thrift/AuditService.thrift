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
    1: string status,
    2: list<string> messages
}

exception AuditServiceException{
    1: string message
}

service AuditService {

    /**
    * Submits Audit(s) Record
    *
    * @param events List of thirft audit.xml events
    *
    * @throws AuditServiceException
    */
    AuditResponse submitAuditEvents(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventAccesses(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventAuthenticates(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventCreates(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventDeletes(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventExports(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventImports(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventModifies(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventSearchQrys(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventSystemActions(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

    /**
      * Submits Audit(s) Record
  	*
  	* @param events List of thirft audit.xml events
  	*
  	* @throws AuditServiceException
    */
    AuditResponse submitEventUnknowns(1: list<events.AuditEvent> events) throws(1: AuditServiceException ex1)

}