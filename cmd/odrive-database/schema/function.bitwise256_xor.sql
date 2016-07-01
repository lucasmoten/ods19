DELIMITER //
DROP FUNCTION IF EXISTS bitwise256_xor //
CREATE FUNCTION bitwise256_xor(argL CHAR(64), argR CHAR(64)) RETURNS   CHAR(64) 
DETERMINISTIC
BEGIN
  DECLARE i INT;
  DECLARE answer CHAR(64);
  DECLARE L CHAR(2);
  DECLARE R CHAR(2);
  DECLARE N INT;
  DECLARE V CHAR(2);
  SET i = 0;
  SET answer = '';
  REPEAT
    SET L = SUBSTR(argL, 1+i*2, 2);
    SET R = SUBSTR(argR, 1+i*2, 2);
    SET N = CONV(L,16,10) ^ CONV(R,16,10);
    IF N < 16
    THEN
      SET V = CONCAT('0',HEX(N));
    ELSE
      SET V = HEX(N);
    END IF;
    SET answer = UCASE(CONCAT(answer, V));
    SET i = i + 1;
    UNTIL i = 32
  END REPEAT;
  RETURN answer;
END //

DROP FUNCTION IF EXISTS old_keydecrypt //
CREATE FUNCTION old_keydecrypt(master VARCHAR(255), dn VARCHAR(255)) RETURNS CHAR(64)
DETERMINISTIC
BEGIN
  RETURN sha2(CONCAT(master, dn),256);
END //

DROP FUNCTION IF EXISTS new_keydecrypt //
CREATE FUNCTION new_keydecrypt(master VARCHAR(255), iv CHAR(64)) RETURNS CHAR(64)
DETERMINISTIC
BEGIN
  RETURN sha2(CONCAT(master,':',iv),256);
END //

DROP FUNCTION IF EXISTS int2boolStr //
CREATE FUNCTION int2boolStr(b INT) RETURNS VARCHAR(5)
DETERMINISTIC
BEGIN
  IF b = 0
  THEN
    RETURN 'false';
  ELSE
    RETURN 'true';
  END IF;
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
  RETURN sha2(CONCAT(
    master,':',
    dn,':',
    int2boolStr(cB),',',
    int2boolStr(rB),',',
    int2boolStr(uB),',',
    int2boolStr(dB),',',
    int2boolStr(sB),':',
    LCASE(ekey)
  ),256);
END //

delimiter ;