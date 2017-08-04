
ALTER TABLE acm	DROP FOREIGN KEY fk_acm_createdBy; 
ALTER TABLE acm	DROP FOREIGN KEY fk_acm_deletedBy;
ALTER TABLE acm	DROP FOREIGN KEY fk_acm_modifiedBy;

ALTER TABLE acmgrantee DROP FOREIGN KEY fk_acmgrantee_userDistinguishedName;
# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE acmgrantee DROP FOREIGN KEY fk_acmgrantee_userdistinguishedname;

ALTER TABLE acmkey DROP FOREIGN KEY fk_acmkey_createdBy;
ALTER TABLE acmkey DROP FOREIGN KEY fk_acmkey_deletedBy;
ALTER TABLE acmkey DROP FOREIGN KEY fk_acmkey_modifiedBy;

ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_createdBy; 
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_deletedBy;
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_modifiedBy;
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_acmId;
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_acmKeyId;
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_acmValueId;

# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE acmpart2 DROP FOREIGN KEY fk_acmpart2_acmid;
ALTER TABLE acmpart2 DROP FOREIGN KEY fk_acmpart2_acmkeyid;
ALTER TABLE acmpart2 DROP FOREIGN KEY fk_acmpart2_acmvalueid;

ALTER TABLE acmvalue DROP FOREIGN KEY fk_acmvalue_createdBy;
ALTER TABLE acmvalue DROP FOREIGN KEY fk_acmvalue_deletedBy;
ALTER TABLE acmvalue DROP FOREIGN KEY fk_acmvalue_modifiedBy;

ALTER TABLE object DROP FOREIGN KEY fk_object_createdBy;
ALTER TABLE object DROP FOREIGN KEY fk_object_deletedBy;
ALTER TABLE object DROP FOREIGN KEY fk_object_expungedBy;
ALTER TABLE object DROP FOREIGN KEY fk_object_modifiedBy;
ALTER TABLE object DROP FOREIGN KEY fk_object_ownedBy;
ALTER TABLE object DROP FOREIGN KEY fk_object_ownedByNew;
ALTER TABLE object DROP FOREIGN KEY fk_object_parentId;
ALTER TABLE object DROP FOREIGN KEY fk_object_typeId;
# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE object DROP FOREIGN KEY fk_object_acmid;
ALTER TABLE object DROP FOREIGN KEY fk_object_createdby;
ALTER TABLE object DROP FOREIGN KEY fk_object_deletedby;
ALTER TABLE object DROP FOREIGN KEY fk_object_expungedby;
ALTER TABLE object DROP FOREIGN KEY fk_object_modifiedby;
ALTER TABLE object DROP FOREIGN KEY fk_object_ownedbyid;
ALTER TABLE object DROP FOREIGN KEY fk_object_parentid;
ALTER TABLE object DROP FOREIGN KEY fk_object_typeid;

ALTER TABLE objectacm DROP FOREIGN KEY fk_objectacm_createdBy;
ALTER TABLE objectacm DROP FOREIGN KEY fk_objectacm_deletedBy;
ALTER TABLE objectacm DROP FOREIGN KEY fk_objectacm_modifiedBy;
ALTER TABLE objectacm DROP FOREIGN KEY fk_objectacm_objectId;
ALTER TABLE objectacm DROP FOREIGN KEY fk_objectacm_acmId;

ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_createdBy;
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_deletedBy;
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_grantee;
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_modifiedBy;
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_objectId;
# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_createdby;
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_createdbyid;
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_grantee;
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_granteeid;
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_objectid;

ALTER TABLE object_property DROP FOREIGN KEY fk_object_property_objectId;
ALTER TABLE object_property DROP FOREIGN KEY fk_object_property_propertyId;
# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE object_property DROP FOREIGN KEY fk_object_property_objectid;

ALTER TABLE object_tag DROP FOREIGN KEY fk_object_tag_createdBy;
ALTER TABLE object_tag DROP FOREIGN KEY fk_object_tag_deletedBy;
ALTER TABLE object_tag DROP FOREIGN KEY fk_object_tag_modifiedBy;
ALTER TABLE object_tag DROP FOREIGN KEY fk_object_tag_objectId;

ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_createdBy;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_deletedBy;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_modifiedBy;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_ownedBy;
# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_createdby;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_deletedby;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_modifiedby;

ALTER TABLE object_type_property DROP FOREIGN KEY fk_object_type_property_propertyId;
ALTER TABLE object_type_property DROP FOREIGN KEY fk_object_type_property_typeId;
# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE object_type_property DROP FOREIGN KEY fk_object_type_property_typeid;

ALTER TABLE property DROP FOREIGN KEY fk_property_createdBy;
ALTER TABLE property DROP FOREIGN KEY fk_property_deletedBy;
ALTER TABLE property DROP FOREIGN KEY fk_property_modifiedBy;
# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE property DROP FOREIGN KEY fk_property_createdby;
ALTER TABLE property DROP FOREIGN KEY fk_property_deletedby;
ALTER TABLE property DROP FOREIGN KEY fk_property_modifiedby;

ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_createdBy;
ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_deletedBy;
ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_modifiedBy;
ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_sourceId;
ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_targetId;

# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE useracm DROP FOREIGN KEY fk_useracm_acmid;
ALTER TABLE useracm DROP FOREIGN KEY fk_useracm_userid;

# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE useraocache DROP FOREIGN KEY fk_useraocache_userid;

# Renames for case sensitivity and new columns in 20170630, must be dropped for force init to drop table
ALTER TABLE useraocachepart DROP FOREIGN KEY fk_useraocachepart_userid;
ALTER TABLE useraocachepart DROP FOREIGN KEY fk_useraocachepart_userkeyid;
ALTER TABLE useraocachepart DROP FOREIGN KEY fk_useraocachepart_uservalueid;

ALTER TABLE user_object_favorite DROP FOREIGN KEY fk_user_object_favorite_createdBy;
ALTER TABLE user_object_favorite DROP FOREIGN KEY fk_user_object_favorite_deletedBy;
ALTER TABLE user_object_favorite DROP FOREIGN KEY fk_user_object_favorite_objectId;

ALTER TABLE user_object_subscription DROP FOREIGN KEY fk_user_object_subscription_createdBy;
ALTER TABLE user_object_subscription DROP FOREIGN KEY fk_user_object_subscription_deletedBy;
ALTER TABLE user_object_subscription DROP FOREIGN KEY fk_user_object_subscription_objectId;