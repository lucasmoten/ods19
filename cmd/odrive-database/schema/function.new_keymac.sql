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
END;
