DELIMITER //
DROP PROCEDURE IF EXISTS rotate_keys //
CREATE PROCEDURE rotate_keys(IN oldMasterPass VARCHAR(255), IN newMasterPass VARCHAR(255), IN entropy VARCHAR(255))
BEGIN 
#slightly dicey.... same IV, new key.  ok for now because underlying file keys being encrypted don't change,
#while the masterkey does change.  so we follow the rule of not repeating same iv under a key againt different data.
#If I could assign a randomm new iv but use the old one to plugin to one part and new one to plug into new part,
#that would let me also change iv here.
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
END //
DELIMITER ;