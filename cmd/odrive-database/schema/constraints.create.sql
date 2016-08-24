SET @OLD_SQL_MODE=@@SQL_MODE;
SET SESSION SQL_MODE='ANSI';

ALTER TABLE acm
	ADD CONSTRAINT fk_acm_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acm_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acm_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
;

ALTER TABLE acmgrantee
    ADD CONSTRAINT fk_acmgrantee_userDistinguishedName FOREIGN KEY (userDistinguishedName) REFERENCES user(distinguishedName)
;

ALTER TABLE acmkey
	ADD CONSTRAINT fk_acmkey_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmkey_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmkey_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
;

ALTER TABLE acmpart
	ADD CONSTRAINT fk_acmpart_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmpart_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmpart_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmpart_acmId FOREIGN KEY (acmId) REFERENCES acm(id)
	,ADD CONSTRAINT fk_acmpart_acmKeyId FOREIGN KEY (acmKeyId) REFERENCES acmkey(id)
	,ADD CONSTRAINT fk_acmpart_acmValueId FOREIGN KEY (acmValueId) REFERENCES acmvalue(id)
;

ALTER TABLE acmvalue
	ADD CONSTRAINT fk_acmvalue_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmvalue_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmvalue_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
;

ALTER TABLE object
	ADD CONSTRAINT fk_object_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_expungedBy FOREIGN KEY (expungedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_ownedBy FOREIGN KEY (ownedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_ownedByNew FOREIGN KEY (ownedByNew) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_parentId FOREIGN KEY (parentId) REFERENCES object(id)
	,ADD CONSTRAINT fk_object_typeId FOREIGN KEY (typeId) REFERENCES object_type(id)
;

ALTER TABLE objectacm
	ADD CONSTRAINT fk_objectacm_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_objectacm_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_objectacm_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_objectacm_objectId FOREIGN KEY (objectId) REFERENCES object(id)
	,ADD CONSTRAINT fk_objectacm_acmId FOREIGN KEY (acmId) REFERENCES acm(id)
;

ALTER TABLE object_permission
	ADD CONSTRAINT fk_object_permission_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_permission_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_permission_grantee FOREIGN KEY (grantee) REFERENCES acmgrantee(grantee)
	,ADD CONSTRAINT fk_object_permission_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_permission_objectId FOREIGN KEY (objectId) REFERENCES object(id)
;

ALTER TABLE object_property 
	ADD CONSTRAINT fk_object_property_objectId FOREIGN KEY (objectId) REFERENCES object(id)
	,ADD CONSTRAINT fk_object_property_propertyId FOREIGN KEY (propertyId) REFERENCES property(id)
;

ALTER TABLE object_tag
	ADD CONSTRAINT fk_object_tag_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_tag_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_tag_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_tag_objectId FOREIGN KEY (objectId) REFERENCES object(id)
;

ALTER TABLE object_type
	ADD CONSTRAINT fk_object_type_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_type_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_type_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_type_ownedBy FOREIGN KEY (ownedBy) REFERENCES user(distinguishedName)
;

ALTER TABLE object_type_property
	ADD CONSTRAINT fk_object_type_property_propertyId FOREIGN KEY (propertyId) REFERENCES property(id)
	,ADD CONSTRAINT fk_object_type_property_typeId FOREIGN KEY (typeId) REFERENCES object_type(id)
;

ALTER TABLE property
	ADD CONSTRAINT fk_property_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_property_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_property_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
;

ALTER TABLE relationship
	ADD CONSTRAINT fk_relationship_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_relationship_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_relationship_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_relationship_sourceId FOREIGN KEY (sourceId) REFERENCES object(id)
	,ADD CONSTRAINT fk_relationship_targetId FOREIGN KEY (targetId) REFERENCES object(id)
;

ALTER TABLE user_object_favorite
	ADD CONSTRAINT fk_user_object_favorite_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_user_object_favorite_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_user_object_favorite_objectId FOREIGN KEY (objectId) REFERENCES object(id)
;

ALTER TABLE user_object_subscription
	ADD CONSTRAINT fk_user_object_subscription_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_user_object_subscription_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_user_object_subscription_objectId FOREIGN KEY (objectId) REFERENCES object(id)
;

SET SESSION SQL_MODE=@OLD_SQL_MODE;
