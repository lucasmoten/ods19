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
END;
