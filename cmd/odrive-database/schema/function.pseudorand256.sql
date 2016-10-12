CREATE FUNCTION pseudorand256(entropy VARCHAR(255)) RETURNS CHAR(64)
BEGIN
  return sha2(CONCAT(NOW(), RAND(), entropy),256);
END;
