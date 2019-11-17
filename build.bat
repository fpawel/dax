SET APP_DIR=build
SET GOOS=windows
SET GOARCH=386
rem SET CGO_ENABLED=0
rem go build -o %APP_DIR%\prog.exe github.com/fpawel/dax/cmd/prog
SET CGO_ENABLED=1
buildmingw32 go build -o %APP_DIR%\dax.exe github.com/fpawel/dax/cmd/dax