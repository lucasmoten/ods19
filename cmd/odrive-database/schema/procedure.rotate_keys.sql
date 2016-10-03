CREATE PROCEDURE rotate_keys(IN oldMasterPass VARCHAR(255), IN newMasterPass VARCHAR(255), IN entropy VARCHAR(255))
BEGIN 
declare exit handler for sqlexception
begin
  rollback;
end;
declare exit handler for sqlwarning
begin
  rollback;
end;
start transaction;
update object_permission p set encryptKey = unhex( 
  bitwise256_xor( 
    new_keydecrypt( newMasterPass, hex(p.permissionIV)), 
    bitwise256_xor(
      new_keydecrypt(oldMasterPass, hex(p.permissionIV)), 
      hex(p.encryptKey)
    )
  ) 
);
update object_permission p set permissionMAC = unhex( new_keymac(newMasterPass, p.grantee, p.allowCreate, p.allowRead, p.allowUpdate, p.allowDelete, p.allowShare, hex(encryptKey)));
commit;
END;