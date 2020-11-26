set GOARCH=386
REM arm
set GOOS=linux
REM windows
REM set GOARM=5
go build -ldflags "-s -w"