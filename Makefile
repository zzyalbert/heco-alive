
hecomon-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/hecomon-linux-amd64 hecomon.go
hecomon:
	go build -o bin/hecomon hecomon.go
geth:
	go build -o bin/geth mock/geth.go

test: geth hecomon
	mv ./bin/geth ./test 
	mv ./bin/hecomon ./test