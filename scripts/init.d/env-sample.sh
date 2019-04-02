#!/bin/bash
export OD_BASEPATH=/opt/services/object-drive-0.0
export OD_LOG_LOCATION=${OD_BASEPATH}/log/object-drive.log
export OD_CACHE_ROOT=${OD_BASEPATH}/cache
export OD_DB_CONN_PARAMS="parseTime=true&collation=utf8_unicode_ci&readTimeout=30s"
export OD_SERVER_CIPHERS='TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA'
