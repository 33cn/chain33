@echo on

set BUILDTIME=%date:~3%-%time:~0,2%:%time:~3,2%:%time:~6,2%
echo %BUILDTIME%

for /F "delims=" %%i in ('git rev-parse --short HEAD') do ( set commitid=%%i)

set BUILD_FLAGS=" -X github.com/33cn/chain33/common/version.GitCommit=%commitid% -X github.com/33cn/chain33/common/version.BuildTime=%BUILDTIME% -w -s"

go env -w CGO_ENABLED=1
go build  -ldflags  %BUILD_FLAGS% -v -o build/chain33.exe github.com/33cn/chain33/cmd/chain33
go build  -ldflags  %BUILD_FLAGS% -v -o build/chain33-cli.exe github.com/33cn/chain33/cmd/cli
