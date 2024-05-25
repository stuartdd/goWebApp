module github.com/stuartdd/goWebApp/server

go 1.22.0

replace github.com/stuartdd/goWebApp/config => ../config

replace github.com/stuartdd/goWebApp/logging => ../logging

replace github.com/stuartdd/goWebApp/controllers => ../controllers

replace github.com/stuartdd/goWebApp/runCommand => ../runCommand

require (
	github.com/stuartdd/goWebApp/config v0.0.0
	github.com/stuartdd/goWebApp/controllers v0.0.0
	github.com/stuartdd/goWebApp/logging v0.0.0
	github.com/stuartdd/goWebApp/runCommand v0.0.0
)
