
ALTER TABLE acm	DROP FOREIGN KEY fk_acm_createdBy; 
ALTER TABLE acm	DROP FOREIGN KEY fk_acm_deletedBy;
ALTER TABLE acm	DROP FOREIGN KEY fk_acm_modifiedBy;

ALTER TABLE acmgrantee DROP FOREIGN KEY fk_acmgrantee_userDistinguishedName;

ALTER TABLE acmkey DROP FOREIGN KEY fk_acmkey_createdBy;
ALTER TABLE acmkey DROP FOREIGN KEY fk_acmkey_deletedBy;
ALTER TABLE acmkey DROP FOREIGN KEY fk_acmkey_modifiedBy;

ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_createdBy; 
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_deletedBy;
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_modifiedBy;
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_acmId;
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_acmKeyId;
ALTER TABLE acmpart DROP FOREIGN KEY fk_acmpart_acmValueId;

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

ALTER TABLE object_property DROP FOREIGN KEY fk_object_property_objectId;
ALTER TABLE object_property DROP FOREIGN KEY fk_object_property_propertyId;

ALTER TABLE object_tag DROP FOREIGN KEY fk_object_tag_createdBy;
ALTER TABLE object_tag DROP FOREIGN KEY fk_object_tag_deletedBy;
ALTER TABLE object_tag DROP FOREIGN KEY fk_object_tag_modifiedBy;
ALTER TABLE object_tag DROP FOREIGN KEY fk_object_tag_objectId;

ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_createdBy;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_deletedBy;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_modifiedBy;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_ownedBy;

ALTER TABLE object_type_property DROP FOREIGN KEY fk_object_type_property_propertyId;
ALTER TABLE object_type_property DROP FOREIGN KEY fk_object_type_property_typeId;

ALTER TABLE property DROP FOREIGN KEY fk_property_createdBy;
ALTER TABLE property DROP FOREIGN KEY fk_property_deletedBy;
ALTER TABLE property DROP FOREIGN KEY fk_property_modifiedBy;

ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_createdBy;
ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_deletedBy;
ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_modifiedBy;
ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_sourceId;
ALTER TABLE relationship DROP FOREIGN KEY fk_relationship_targetId;

ALTER TABLE user_object_favorite DROP FOREIGN KEY fk_user_object_favorite_createdBy;
ALTER TABLE user_object_favorite DROP FOREIGN KEY fk_user_object_favorite_deletedBy;
ALTER TABLE user_object_favorite DROP FOREIGN KEY fk_user_object_favorite_objectId;

ALTER TABLE user_object_subscription DROP FOREIGN KEY fk_user_object_subscription_createdBy;
ALTER TABLE user_object_subscription DROP FOREIGN KEY fk_user_object_subscription_deletedBy;
ALTER TABLE user_object_subscription DROP FOREIGN KEY fk_user_object_subscription_objectId;


