DELIMITER //

DROP FUNCTION IF EXISTS old_keydecrypt //
CREATE FUNCTION old_keydecrypt(master VARCHAR(255), dn VARCHAR(255)) RETURNS CHAR(64)
BEGIN
  RETURN sha2(CONCAT(master, dn),256);
END //

DROP FUNCTION IF EXISTS new_keydecrypt //
CREATE FUNCTION new_keydecrypt(master VARCHAR(255), iv CHAR(64)) RETURNS CHAR(64)
BEGIN
  RETURN sha2(CONCAT(master,':',LCASE(iv)),256);
END //

DROP FUNCTION IF EXISTS int2boolStr //
CREATE FUNCTION int2boolStr(b INT) RETURNS VARCHAR(5)
BEGIN
  IF b = 0
  THEN
    RETURN 'false';
  ELSE
    RETURN 'true';
  END IF;
END //

#stupid, but maybe no alt right now - if rand is weak,
#then NOW() is of little protection because of timestamps
DROP FUNCTION IF EXISTS pseudorand256 //
CREATE FUNCTION pseudorand256(entropy VARCHAR(255)) RETURNS CHAR(64)
BEGIN
  return sha2(CONCAT(NOW(), RAND(), entropy),256);
END //

DROP FUNCTION IF EXISTS new_keymacdata //
CREATE FUNCTION new_keymacdata(
  master VARCHAR(255), 
  dn VARCHAR(255),
  cB INT,
  rB INT,
  uB INT,
  dB INT,
  sB INT,  
  ekey CHAR(64)) RETURNS VARCHAR(255)
BEGIN
  RETURN CONCAT(
    master,':',
    dn,':',
    int2boolStr(cB),',',
    int2boolStr(rB),',',
    int2boolStr(uB),',',
    int2boolStr(dB),',',
    int2boolStr(sB),':',
    LCASE(ekey)
  );
END //


DROP FUNCTION IF EXISTS new_keymac //
CREATE FUNCTION new_keymac(
  master VARCHAR(255), 
  dn VARCHAR(255),
  cB INT,
  rB INT,
  uB INT,
  dB INT,
  sB INT,  
  ekey CHAR(64)) RETURNS CHAR(64)
DETERMINISTIC
BEGIN
  RETURN sha2(new_keymacdata(
    master,
    dn,
    cB,
    rB,
    uB,
    dB,
    sB,
    LCASE(ekey)
  ),256);
END //

DELIMITER ;