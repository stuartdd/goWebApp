module stuartdd.com/server

go 1.22.0

replace stuartdd.com/config => ../config

replace stuartdd.com/logging => ../logging

replace stuartdd.com/controllers => ../controllers

replace stuartdd.com/runCommand => ../runCommand

require (
	stuartdd.com/config v0.0.0-00010101000000-000000000000
	stuartdd.com/controllers v0.0.0
	stuartdd.com/logging v0.0.0
)

require stuartdd.com/runCommand v0.0.0 // indirect
