CREATE PROCEDURE migrate_keys(IN masterPass VARCHAR(255), IN entropy VARCHAR(255))
BEGIN 
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
END;


