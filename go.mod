module github.com/stuartdd/goWebApp

go 1.22.0

replace stuartdd.com/config => ./config

replace stuartdd.com/server => ./server

replace stuartdd.com/logging => ./logging

replace stuartdd.com/controllers => ./controllers

replace stuartdd.com/runCommand => ./runCommand

replace stuartdd.com/pictures => ./pictures

replace stuartdd.com/image => ./image

require (
	stuartdd.com/config v0.0.0-00010101000000-000000000000
	stuartdd.com/server v0.0.0-00010101000000-000000000000

)

require (
	stuartdd.com/controllers v0.0.0 // indirect
	stuartdd.com/logging v0.0.0
	stuartdd.com/pictures v0.0.0
	stuartdd.com/runCommand v0.0.0 // indirect
)
