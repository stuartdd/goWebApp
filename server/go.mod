module github.com/stuartdd/goWebApp/server

go 1.22.0

replace github.com/stuartdd/goWebApp/config => ../config

replace github.com/stuartdd/goWebApp/logging => ../logging

replace github.com/stuartdd/goWebApp/controllers => ../controllers

replace github.com/stuartdd/goWebApp/runCommand => ../runCommand

require (
	github.com/stuartdd/goWebApp/config v1.0.0
	github.com/stuartdd/goWebApp/controllers v1.0.0
	github.com/stuartdd/goWebApp/logging v1.0.0
	github.com/stuartdd/goWebApp/runCommand v1.0.0
)

require (
	github.com/stuartdd/goWebApp/image v1.0.0 // indirect
	github.com/stuartdd/goWebApp/pictures v1.0.0 // indirect
)

replace github.com/stuartdd/goWebApp/image v1.0.0 => ../image

replace github.com/stuartdd/goWebApp/pictures v1.0.0 => ../pictures
