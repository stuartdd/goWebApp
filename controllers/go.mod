module github.com/stuartdd/goWebApp/controllers

go 1.22.0

replace github.com/stuartdd/goWebApp/runCommand => ../runCommand

replace github.com/stuartdd/goWebApp/config => ../config

replace github.com/stuartdd/goWebApp/pictures v1.0.0 => ../pictures

require github.com/stuartdd/goWebApp/runCommand v1.0.0

require github.com/stuartdd/goWebApp/config v1.0.0

require github.com/stuartdd/goWebApp/pictures v1.0.0

require github.com/stuartdd/goWebApp/image v1.0.0 // indirect

replace github.com/stuartdd/goWebApp/image v1.0.0 => ../image
