DELIMITER //

# We ensure that rewrites of encrypt key happen in one statement across all objects, as
# a partial write of the keys will lose data.
DROP PROCEDURE IF EXISTS migrate_keys //
CREATE PROCEDURE migrate_keys(IN masterPass VARCHAR(255), IN entropy VARCHAR(255))
BEGIN 
#passing in id makes it unique just in case rand repeats and we are still at same time in ms
update object_permission p set permissionIV = unhex(pseudorand256(concat(entropy,hex(p.id))));
update object_permission p set encryptKey = unhex( 
  bitwise256_xor( 
    new_keydecrypt( masterPass, hex(p.permissionIV)), 
    bitwise256_xor( 
      old_keydecrypt(masterPass, p.grantee), 
      hex(p.encryptKey)
    )
  )
);
update object_permission p set permissionMAC = unhex( new_keymac(masterPass, p.grantee, p.allowCreate, p.allowRead, p.allowUpdate, p.allowDelete, p.allowShare, hex(encryptKey)));
END //

DELIMITER ;

