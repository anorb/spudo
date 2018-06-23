find -name "*.go" ! -name "*_test.go" -exec go build -buildmode=plugin {} \;
