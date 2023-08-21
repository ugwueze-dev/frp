@echo off

set AUTH_URL="http://{ipaddress}/api/webserver/SesTokInfo"

curl %AUTH_URL% > temp.txt

set /p $RESPONSE = < temp.txt


for /f "tokens=2 delims=^<^>" %%a in (temp.txt) do (
  set "$session_id=%%a"
goto:next)

:next
for /f "skip=3 tokens=2 delims=^<^>" %%a in (temp.txt) do (
  set "$token=%%a"
goto:end)

:end
curl -X POST http://{ipaddress}/api/device/control -H "Cookie: %$session_id%" -H "__RequestVerificationToken: %$token%" --data "<?xml version=\"1.0\" encoding=\"UTF-8\"?><request><Control>1</Control></request>"
