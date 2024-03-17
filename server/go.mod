module stuartdd.com/server

go 1.22.0

replace stuartdd.com/config => ../config

replace stuartdd.com/tools => ../tools

replace stuartdd.com/controllers => ../controllers

replace stuartdd.com/runCommand => ../runCommand

require (
	stuartdd.com/config v0.0.0-00010101000000-000000000000
	stuartdd.com/controllers v0.0.0
	stuartdd.com/tools v0.0.0-00010101000000-000000000000
)

require stuartdd.com/runCommand v0.0.0 // indirect
