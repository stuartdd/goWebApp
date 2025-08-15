# Server Documentation:

## Users

Config:__"UserDataPath"__ and __"Users"__

Config-Section:__"Users"__ defines a list of users for the system. 

Config:__"UserDataPath"__ defines an external file that contains a list of users. This will be merged with __"Users"__ when looaded. The format of the external file is the same as the __"Users"__ section.

``` golang
type UserData struct {
	Hidden    *bool             // If true the user will not appear in the users list "http://server:port/server/users"
	Name      string            // The name of the user. If the user ID is bob. The name could be Bob. 
	Home      string            // All locations are prefixed with this path when resolved
	Locations map[string]string // Name,Value list for locations. The names are public the values are resolved relative to Home
	Env       map[string]string // Name,Value list combined with OS environment for substitutions in resolved locations
}
```
- All resources are with respect to a user.
- All file locations must be defined on the Config:"Users":user:"Loc" section
- If Config:"Users":user:"Home" is not provided then the users ID is substituted.

## Locations

Each used has a list of 'Locations'. These are resolved when the server starts.

### Example
```
"Users": {
  "bert": {
    "Hidden": null,
    "Name": "Bertrand",
    "Home": "",
    "Locations": {
      "data": "stateData",
      "original": "***%{WebServerPictures}/originals/%{id}"
    },
    "imageRoot": "%{WebServerPictures}/originals",
    "imagePaths": [
      "Bobs_Phone",
      "WhatsApp"
    ]
	,,,
}
```
The user id = "bert"
The users display name is "Bertrand"
The user is NOT hidden
The user "Home" is defined by the user id

Config:"ServerDataRoot" = "/home/goWebApp/server/usersdata/"
Config:"Users":bert:"Home" = 'bert' (as it is undefined)
The environment variable "WebServerPictures" = "/media/pictures/"

Then:
Config:"Users":bert:"Locations":data resolves to: /home/goWebApp/server/usersdata/bert/stateData
Config:"Users":bert:"Locations":data resolves to: /media/pictures/originals/bert

Note that "%{id}" is replaced with the user id (bert).

### *** prefix

The "***" prefix on a  "original" location forces an 'Absolute' resolution of a path. 
- It is NOT relative to Config:"ServerDataRoot" and  Config:"Users":bert:"Home"
- In this example the path is defined by the environment variable "WebServerPictures"
- The `"***"` must be followed by a `/`. In the above example this is defined in environment variable  "WebServerPictures"

Config:"Users":bert:"Home" can also have a `"***"` prefix.
- For example if "Users":bert:"Home" = "***/media/%{id}"
- Location 'data' would resolve to "/media/bert/stateData"
- Location 'original' would still resolve to "/media/pictures/originals/bert" as it is already absolute.

#### Warning

__Minimal use of '***' is recommended as it allows files 'outside' the server environment to be accessed.__

### User:Env

The name value pairs defined in Config:"Users":bert:"Env" are available for substitution in user 'Locations' in the same way as %{id}

### Additional data

In the above example the 'imageRoot' and 'imagePaths' are ignored by the server. 

The main reason for externaising the user data is that it can be read by other server related applications. 

'imageRoot' and 'imagePaths' are both used by the Thumbnail image generator.

