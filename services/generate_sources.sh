#!/bin/bash

audit_prefix="decipher.com/object-drive-server/services/audit/generated/"

generator -go.signedbytes=true -go.importprefix=$audit_prefix ./audit/thrift/AuditService.thrift ./audit/generated


