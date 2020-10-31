set GOARCH=386
REM arm
set GOOS=windows
REM linux
REM set GOARM=5
go build -ldflags "-s -w"