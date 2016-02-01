/* Constants used . */

namespace * gov.ic.dodiis.dctc.bedrock.audit.thrift

enum Type {
	EventAccess,
	EventAuthenticate,
	EventCreate,
	EventDelete,
	EventExport,
	EventImport,
	EventModify,
	EventSearchQry,
	EventSystemAction,
	EventUnknown
}

enum Action {
	ACCESS,
	ACCESS_DECISION,
	ACTIVATE,
	ADMIN_ROOT_ACCESS,
	AUTHENTICATE,
	CLEAR,
	CREATE,
	DEACTIVATE,
	DELETE,
	EXPORT,
	IMPORT,
	INSERT,
	LOGIN,
	LOGOUT,
	MODIFY,
	MODIFY_CONFIG,
	MODIFY_POLICY,
	MOVE,
	OWNERSHIP_MODIFY,
	PERMISSION_MODIFY,
	PRINT,
	REMOVE,
	RESTART,
	ROLE_ESCALATION,
	SEARCH,
	SHUTDOWN,
	UNLOCK,
	USRGRP_ADD,
	USRGRP_DELETE,
	USRGRP_LOCK,
	USRGRP_MODIFY,
	USRGRP_SUSPEND
}

enum ActionResult {
    DENY,
    FAILURE,
    GRANT,
    SUCCESS
}

enum ActionMode {
    ADD_CHANGE_REQUEST_FORM,
    ADMIN_REQUEST,
    SESSION_TIMEOUT,
    SYSTEM_FORCED_LOGOFF,
    SYSTEM_TIMEOUT,
    USER_INITIATED,
    USER_REQUEST,
    UNKNOWN
}

enum ResultSet {
    FILE,
    OBJECT,
    OTHER
}

enum QueryType {
    BROKERED_SEARCH,
    BROWSER,
    SERVICE_DRIVEN,
    USER_DRIVEN,
    UNKNOWN
}