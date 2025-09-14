module github.com/stuartdd/goWebApp

go 1.24.0

replace github.com/stuartdd/goWebApp/config => ./config

replace github.com/stuartdd/goWebApp/server => ./server

replace github.com/stuartdd/goWebApp/logging => ./logging

replace github.com/stuartdd/goWebApp/controllers => ./controllers

replace github.com/stuartdd/goWebApp/runCommand => ./runCommand

require (
	github.com/stuartdd/goWebApp/config v1.0.0
	github.com/stuartdd/goWebApp/logging v1.0.0
	github.com/stuartdd/goWebApp/runCommand v1.0.0
	github.com/stuartdd/goWebApp/server v1.0.0

)

require github.com/stuartdd/goWebApp/controllers v1.0.0 // indirect
