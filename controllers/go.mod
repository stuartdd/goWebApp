module github.com/stuartdd/goWebApp/controllers

go 1.24.0

replace github.com/stuartdd/goWebApp/runCommand => ../runCommand

replace github.com/stuartdd/goWebApp/config => ../config

replace github.com/stuartdd/goWebApp/logging => ../logging

require github.com/stuartdd/goWebApp/runCommand v1.0.0

require github.com/stuartdd/goWebApp/config v1.0.0

require github.com/stuartdd/goWebApp/logging v1.0.0
