echo "building chain33.exe chain33-cli.exe"
$commitid = git rev-parse --short=8 HEAD
echo $commitid

$BUILDTIME=get-date -format "yyyy-MM-dd/HH:mm:ss"
echo $BUILDTIME

$BUILD_FLAGS='''-X "github.com/33cn/chain33/common/version.GitCommit={0}" -X "github.com/33cn/chain33/common/version.BuildTime={1}" -w -s''' -f $commitid,$BUILDTIME
echo $BUILD_FLAGS


go env -w CGO_ENABLED=1
go build  -ldflags  $BUILD_FLAGS  -v -o build/chain33.exe github.com/33cn/chain33/cmd/chain33
go build  -ldflags  $BUILD_FLAGS  -v -o build/chain33-cli.exe github.com/33cn/chain33/cmd/cli

echo "build end"
