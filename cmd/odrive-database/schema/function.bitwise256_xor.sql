CREATE FUNCTION bitwise256_xor(argL CHAR(64), argR CHAR(64)) RETURNS CHAR(64) 
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
END; 