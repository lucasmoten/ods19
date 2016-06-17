
# This should succeed
INSERT INTO object_type (name, createdBy) values ('file','Bob');
DELETE from object_type;
DELETE from a_object_type;
UPDATE object_type set isDeleted = 1;
select count(*) from object_type;
select count(*) from a_object_type;

# This should error since an object type of the name exists
INSERT INTO object_type (name, createdBy) values ('file','Bob');
# This should fail because createdBy not specified
INSERT INTO object_type (name) values ('doc');
# This should fail because createdBy column not given in column clause
INSERT INTO object_type (name) values ('doc','Jane');
# This should succeed and mark as owned by Bill
INSERT INTO object_type (name,createdBy,ownedBy) values ('doc','Jane','Bill');
# This should return 2 rows
select id,createdDate,createdBy,modifiedDate,modifiedBy,ownedBy,changeCount,changeToken from object_type;

UPDATE object_type SET modifiedBy = 'Jill', ownedBy = 'Jane' WHERE id = '27c43f1e-9960-11e5-a697-0242ac110002';
UPDATE object_type SET modifiedBy = 'Jill', ownedBy = 'Jane', changeToken = '8414497c68dba7a01d8fdb718d006b12', changeCount = 1 WHERE id = 'ee017550-9958-11e5-a697-0242ac110002';

