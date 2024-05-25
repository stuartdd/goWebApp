module github.com/stuartdd/goWebApp

go 1.22.0

replace github.com/stuartdd/goWebApp/config => ./config

replace github.com/stuartdd/goWebApp/server => ./server

replace github.com/stuartdd/goWebApp/logging => ./logging

replace github.com/stuartdd/goWebApp/controllers => ./controllers

replace github.com/stuartdd/goWebApp/runCommand => ./runCommand

replace github.com/stuartdd/goWebApp/pictures => ./pictures

replace github.com/stuartdd/image => ./image

require (
	github.com/stuartdd/goWebApp/config v1.0.0
	github.com/stuartdd/goWebApp/logging v1.0.0
	github.com/stuartdd/goWebApp/pictures v1.0.0
	github.com/stuartdd/goWebApp/server v1.0.0
)

require (
	github.com/stuartdd/goWebApp/controllers v1.0.0 // indirect
	github.com/stuartdd/goWebApp/runCommand v1.0.0 // indirect
)
