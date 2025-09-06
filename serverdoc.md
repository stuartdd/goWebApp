# Server Documentation:
## Config File

Definitions of configuration data in this doc are defined using the following term:

__Config:"Users"__ refers to the main JSON __Config:__ file, root element __Users__.

__Config:"Users"__:bert:__Locations__ refers to the main JSON __Config:__ file, root element, called __Users__ with an entry (map name) "bert" with the sub element __Locations__.

### Validation

All file paths defined in the configuration data will be resolved and the resulting path checked. 

Paths that do not exist will cause configuration validation to fail.

The final configuration will be echoed to the console (and the log) if the verbose (-v) option is provided. Use test (-t) to prevent the server from starting.

The server will not start if there are errors.

If the -c option is used then User Locations will be created before the server starts.

## Users

__Config:"UserDataPath"__ and __Config:"Users"__

__Config:"Users"__ defines a list of users for the system. 

__Config:"UserDataPath"__ defines an external file that contains a list of users. This will be merged with __Config:"Users"__ when loaded. 

The format of the external file is the same as the __Config:"Users"__  section.

``` golang
type UserData struct {
	Hidden    *bool             // If true the user will not appear in the users list "http://server:port/server/users"
	Name      string            // The name of the user. If the user ID is bob. The name could be Bob. 
	Home      string            // All locations are prefixed with this path when resolved
	Locations map[string]string // Name,Value list for locations. The names are public the values are resolved relative to Home
	Env       map[string]string // Name,Value list combined with OS environment for substitutions in resolved locations
}
```
- All locations are with respect to a user id.
- All user file locations for user id 'bert' must be defined in the __Config:"Users"__:bert:__"Loc"__ section
- If __Config:"Users"__:bert:__"Home"__ is not defined then the users ID is substituted.

#### Example (Main Config)
The folllowing is a user with a user id of 'bert' and a name 'Bertrand', as defined in the main configuration file.

Additional users can be defined in an external file. For example the file __'userData.json'__ could contain many additional users. If __Config:"UserDataPath"__ is defined and the file contains users then these will be merged in to __Config:"Users"__. Duplicate user id values will cause config validation to fail.

The content of __'userData.json'__ is the same as __Config:"Users"__ without the "Users" map label.

```
"UserDataPath":"%{WebServerData}/userData.json",
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
- The user id = "bert"
- The users display name is "Bertrand"
- The user is NOT hidden
- The user "Home" is defined by the user id

When:

- __Config:"ServerDataRoot"__ = "/home/goWebApp/server/usersdata/"
- __Config:"Users"__:bert:__"Home"__ = 'bert' (as it is undefined)
- The environment variable "WebServerPictures" = "/media/pictures/"

Then:

- __Config:"Users"__:bert:"Locations":data resolves to: /home/goWebApp/server/usersdata/bert/stateData
- __Config:"Users"__:bert:"Locations":data resolves to: /media/pictures/originals/bert

Note that "%{id}" is replaced with the user id (bert in this example).

### Locations

Each used has a list of 'Locations'. These are resolved when the server starts.

### *** prefix

The "***" prefix on a  "original" location forces an 'Absolute' resolution of a path. 
- It is NOT relative to __Config:"ServerDataRoot"__ and  __Config:"Users"__:bert:__"Home"__
- In this example the path is defined by the environment variable "WebServerPictures"
- The `"***"` must be followed by a `/`. In the above example this is defined in environment variable  "WebServerPictures"

__Config:"Users"__:bert:__"Home"__ can also have a `"***"` prefix.
- For example if "Users":bert:__"Home"__ = "***/media/%{id}"
- Location 'data' would resolve to "/media/bert/stateData"
- Location 'original' would still resolve to "/media/pictures/originals/bert" as it is already absolute.

#### Warning

__Minimal use of '***' is recommended as it allows files 'outside' the server environment to be accessed.__

### User:Env

The name value pairs defined in __Config:"Users"__:bert:__"Env"__ are available for substitution in user 'Locations' in the same way as %{id}

### Additional data

In the above example the 'imageRoot' and 'imagePaths' are ignored by the server. 

The main reason for externalising the user data is that it can be read by other server related applications.

'imageRoot' and 'imagePaths' are both used by the Thumbnail image generator, they are ignored by the server.

