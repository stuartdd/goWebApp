module github.com/stuartdd/goWebApp

go 1.22.0

replace stuartdd.com/config => ./config

replace stuartdd.com/server => ./server

replace stuartdd.com/tools => ./tools

replace stuartdd.com/controllers => ./controllers

require (
	stuartdd.com/config v0.0.0-00010101000000-000000000000
	stuartdd.com/server v0.0.0-00010101000000-000000000000
)

require (
	stuartdd.com/controllers v0.0.0-00010101000000-000000000000 // indirect
	stuartdd.com/tools v0.0.0-00010101000000-000000000000 
)
