----------------------------------------
Help For Golang Web Application (Server)
----------------------------------------

Application name: [appName] (this is 'goWebApp' unless changed by the developer) 

Without parameters it will load the default configuration data: [appName].json

The parameter config=[configName] will load an alternative config data file.

Parameters are not case sensitive.

------
Usage:
------

[appName] help 
    This will load the help text (this file) and echo it to the console.

[appName] create
    This will load the configuration file and create any user directories required
    to run. It will only create user directories ('Locations' in the config data).

    Can be used with the "config=" parameter.

    config=goWebApp create

    The application will terminate after the data is created.

[appName] scan [userName]
    This will scan the users 'original' path defined in config json file.
    
    For example 'scan bob'

    Where user 'bob' is defined as follows:

      "bob": {
        "Home": "",
        "Locations": {
          "home": "",
          "original": "bobPictures"
        },
        "Exec": null,
        "Env": null
      } 

  Will scan the 'bobPictures' directory for changes to the file structure.

[appName] add [userName]
    This will add user data to the configuration file. The parameter after the 'add'
    is the user id of the user.

    The following is an example of the data added from:

    [appName] add userx

    "userx": {
      "Hidden": null,
      "Name": "Userx",
      "Home": "userx",
      "Locations": {
        "data": "stateData",
        "home": ""
      },
      "Exec": {},
      "Env": {}
    }

    The user is is always lowercase. The name is created from the user is with the first
    letter capitalised. Wou can manually edit the users date to change the details.

    Running  [appName] create after an add user ll create the required locations.

    Can be used with the "config=" parameter.

    config=goWebApp add userx

    The application will terminate after the data is created.

