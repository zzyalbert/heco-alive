
all:
	go build && mv hecomon test/

hecomon-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/hecomon-linux-amd64 hecomon.go
hecomon:
	go build -o bin/hecomon hecomon.go
