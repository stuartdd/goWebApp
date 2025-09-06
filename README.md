# goWebApp

Web app written in go for raspberry pi home server

## Command line options

### config=```<fileNane>```

Defines the name of the configuration data (described below)

```<fileName>``` is the path to the configuration file.

If the file name is empty then a file name will be derives from the module name (goWebApp) with the '.json' file extension.

### (verbose) -v -vr

The -v options causes the resolved config data and other helpfull diagnostin data to be echoed to the console.

The -v option will NOT start the server. Use -vr for verbose and run the server.

### (kill) -k -kr

The -k options causes the server defined in the config file to 'EXIT' with a return code 11.

This can also be done with the following http request:

```
http://<hostIpAddress>:<hostPort>/server/exit
```

The -k option will NOT start the server. Use -kr for KILL and RUN the server.

A 1 second delay is used after the KILL message is sent to allow the server to close cleanly before it is started.

### Create locations

The ```create``` command line option will create the directories listed in EACH users 'Locations' section of the config file.

When the server is run it will fail to start if the paths do not exist. If the locations cannot be created the server cannot start.

### add user

This will add a user to the configuration file with the default values.

```add userOne``` will add the following user to the config:

```json
"oneUser": {
  "Hidden": null,
  "Name": "OneUser",
  "Home": "",
  "Locations": {
    "data": "stateData"
  },
  "Exec": {},
  "Env": {}
},
```

The directory ```<ServerDataRoot>```/oneUser/stateData will need to be created before the server will start.

Use the 'create' command line option to create the required path.

## Configuration data

### Config file name

The name of the configuration file is defined as follows:

1 First program argument

2 The module name (goWebApp unless altered for a build)

If it does not have '.json' on the end it will be added.

The file must be valid JSON format

Refer to the config.go module for the fine detail: 

Once loaded the fields are parsed and file locations are fixed.

Errors in the data will abort the application.

The following high level values are defiend as:

Note that GO's json Unmarshal (The process that reads the JSON) is not case sensitive for the FIRST character of the name value so **ServerDataRoot** and **serverDataRoot** are equivilant. 

## **port**

Defines the integer port number for the application to run on. For example port 8082 will require the following URL

```
http://localhost:8082/static/dart.html
```

Not ports below 100 require admin privilages to run.

## **ServerDataRoot**

This is the FIXED location of the users data. When users are defined (including the admin user) any 'Locations' defined for the user MUST exist when the server starts up.

A users home directory is defined as a concatenation of the following:

**ServerDataRoot**

**Usera-->username-->Home** 

All locations defined in the config file, except **ServerDataRoot** and **ServerStaticRoot** are prefixed with 
**ServerDataRoot**. Environment values are substituted (Ref Environment Substitution") when the server loads.

## **ServerStaticRoot**

This is the FIXED path to the web applications static files. Environment values are substituted (Ref Environment Substitution") when the server loads.

Html, CSS, JavaScript and image files are usually stored here. They are accesed as fillows:

```
http://localhost:8082/static/dart.html

http://localhost:8082/static/dart.css

http://localhost:8082/static/stuart.png.
```

If images are stored in a sub directory. For example 'images' the path will be followed. 

Keep the dir names simple (without spaces and special characters)

```
http://localhost:8082/static/images/stuart.png.
```

No Environment Substitution takes place on this value. It is fixed onvce the server is running.

It can be anywhere in the file system.

## **ThumbNailTrim**

Thumbnails and the pitures the Thumbnails are derived from have different name formats. 

A picture called MyPic.jpg will have a thumbnail YYYY_MM_DD_HH_MM_SS_MyPic.jpg.jpg. This is so the thumnails are sorted correctly when displayed and they are always jpg files (The picture may not be a jpg).

When a call is made to:

```
/files/user/*/loc/*/path/*/name/*

or 

/files/user/*/loc/*/name/*
```

An additional Query parameter should be added if the request name was a thumbnail and the response name is a picture. For example:

```
/files/user/*/loc/*/path/*/name/*?thumbnail=true

or 

/files/user/*/loc/*/name/*?thumbnail=true
```

This will then use the first value of **ThumbNailTrim** to remove the first n chars and the second value to remove the last n chars.

If **ThumbNailTrim** is undefined the default will be [0,0]. The file name will not change.

If **ThumbNailTrim** has a single value [20] then the first 20 chars are removed from the thumbnail name.

If **ThumbNailTrim** has two values [20, 4] then the first 20 and the last 4 chars are removed thumbnail name.

So 'YYYY_MM_DD_HH_MM_SS_MyPic.jpg.jpg' would look for file 'MyPic.jpg'

## **faviconIcoPath**

Browsers always request the 'favicon' to give the browser tab an icon value. **faviconIcoPath** holds the path to the file and the file name.

This is only valid for requests:

```
http://localhost:8082/favicon.ico
```

This value is prefixed with the value from **ServerDataRoot** and environment values are substituted (Ref Environment Substitution").

## **reloadConfigSeconds**

When a request received by the server, and before the request is submitted, the configuration file __MAY__ reload. 

This allows changes in the config data while the server is running.

The config file will only reload **reloadConfigSeconds** after the previous reload.

## **Users**

Users defines the resources fo a given user (including the admin user).

```json
"Users": {
        "admin": {
        },
        "bob": {
        },
        "stuart": {
        }
    }
```

When a request is received the data is mapped to a '**User**', a '**Location**' and optionally a file name or resource.

Each '**User**' has a set of '**Location**'s as follows:

```json
"stuart": {
   "name": "Stuart",
   "home": "stuartsData",
   "Locations": {
      "home": "",
      "data": "stateData",
      "pics": "pictures"
   }
}
```

Additional elements for User:

**'Hidden'** If true the user is not returned in any lists. This is used for the admin user.

**'Name'** The users proper name.

**'Home'** The home path for the user. This is prefixed with **ServerDataRoot** on load unless prefixed with '***' in which case it is an absolute path.

**'Locations'** See below

**'Exec'** Used to define operating system (external) utilities. See below.

**'Env'** Used to define user specific Environment Substitution values.

### User and Location name resolution

Request ```http://localhost:8082/files/user/stuart/loc/pics/name/stuart.jpeg```

Would be resolved to a file path as follows:

```
**ServerDataRoot**/stuartsData/pictures/stuart.jpeg
```

The '/files/' indicates that this is a file operation.

The '/user/stuart/' element finds stuart in the Users section. The value from 'stuart-->home' is used to build the file path. 

The '/loc/pics/' element finds the 'pics' element in the users Location section. The value from 'stuart-->Locations-->pics' is used to build the file path.

If any of this fails a 404 error is returned.

Note that if the '/user/stuart/' element is undefined then '/user/admin/' is substituted.

This mapping is FIXED when the application loads and Location values are substituted (Ref Environment Substitution") at that time.

The '/name/' section above indicates that a file is requested. In this case 'stuart.jpeg'. If found it is returned.

An additional '/path/pathToFiles/' can be added to locate files in sub directories of **ServerDataRoot**/stuartsData/pictures. 

Note that full paths and file names that are sent in url requests are normally encoded to stop invalid chracters being interpreted incorrectly. If this is the case the path or file name is prefixed with "X0X". For example: 

```
 /files/user/stuart/loc/pics/path/X0Xcy10ZXN0Zm9sZGVy/name/X0XcGljMS5qcGVn.
```

The 'path/X0XVGh1bWJzMDAx/' requests a base64 encoded path. The '/name/X0XcGljMS5qcGVn' requests a base64 encoded file name.

X0Xcy10ZXN0Zm9sZGVy decodes to "s-testfolder" but may include multiple path elements.
X0XcGljMS5qcGVn decoded to "pic1.jpeg"

Note that responses from file lists and path list requests will contain both the display name and the base64 encoded name. For example

```
http://localhost:8082/paths/user/stuart/loc/pics
```

Returned: 

```json
{
   "error":false,
   "user":"stuart",
   "loc":"pics",
   "paths":[{
      "name":"s-testfolder",
      "encName":"X0Xcy10ZXN0Zm9sZGVy"
      },
      {
      "name":"s-testfolder/s-testdir1",
      "encName":"X0Xcy10ZXN0Zm9sZGVyL3MtdGVzdGRpcjE="
      }]
   }
```

When building requests it is best to use the 'encName' values for paths and file names. This is better if the file names have spaces or invalid characters in them. 

```
http://<ServerIpAddress>:<ServerPort>/files/user/stuart/loc/pics/name/X0Xcy10ZXN0Zm9sZGVyL3MtdGVzdGRpcjE=
```

The value 'encName' is the value 'name' base64 encoded and prefixed with 'X0X'. It is recognised by the server if prefixed with 'X0X'.

If the base64 value does not have a X0X then add the query parameter '?base64=true'. If this is done then the 'path' data in any url must also be base64 encoded.

## **Exec**

Each user can have a set of Operating System commands that can be run on request. For example:

```json
"ds": {
   "Cmd": [
      "./diskSize.sh"
    ],
    "Dir": "",
    "StdOutType": "json",
    "Log": "",
    "LogOut": "",
    "LogErr": "",
    "NzCodeReturns": 200,
    "Detached": false
},
```

```
http://localhost:8082/exec/user/stuart/exec/ds
```

This will locate the user 'stuart' and within the **Exec** section will locate the **ds** command. 

It will template the command and all of the command parameters before running the command.

The working directory for the command can be defined by the **Dir** element, This will be the users home directory if undefined. Otherwise it is a path within that directory .

The sysOut stream from the command can be saved in a file using the **Log+OutLog** element as a path.

The sysErr stream from the command can be saved in a file using the **Log+ErrLog** element as a path.

The return code is checked and the response generated.

### Exec Response

```
"nzCodeReturns": 200, TO BE FIXED!
```

If a non zero return code is a success the http status code will be 200. If this is not defined a non zero return code will return 

## **filterFiles**

This restricts the file types that file queries can return. 

```json
"filterFiles": [
        "Json",
        "jpeg",
        "jpg",
        "png",
        "log"
    ]
```

So a request to list files in a directory will be restricted to files with the extenstions listed above.

Also excluded are file names that start with a '.' and an '_'. This applies to directory listings as well.

## **ContentTypeCharset**

When a file is returned it's extension is used to look up the mime type. This 'mime type' is returned in the http header 'ContentType' to tell the browser what is being returned. Json, text a jpeg etc.

In the code mimeTypes.go in the config module a map of mime types is embedded. Here is a small section:

```golang
    mime["jpeg"] = "image/jpeg"
    mime["jpg"] = "image/jpeg"
    mime["js"] = "text/javascript%0"
    mime["json"] = "application/json%0"
```

So a file with an extension of .jpeg is returned with a 'image/jpeg' ContentType.

For text based files it is also important to define the character set. For example utf-8.

For the json extension shown above there is a '%0' marker at the end of the mime type. This is replaces with the character set encoding defined in **ContentTypeCharset**.

If **ContentTypeCharset=utf-8**  a json file will be returned with the following ContentType:

```
application/json; charset=utf-8
```

If **ContentTypeCharset** is undefined a json file will be returned with the following ContentType:

```
application/json
```

## **Env**

This adds Environment Substitution values at a Global level.

When performing Substitution the presedence is as follows. 

1 User 'Env'

2 global 'Env'

3 Operating system environment variables.

```json
"Env": {
   "lsargs": "-l"
}
```

Thia adds the Substitution name (or key) 'lsargs' with a value of '-l'. This will be included in any Run-Time Substitution.

If the same name is defined at the user level and the global level the **user** level wins.

## **LogData**

This defines how the logging data is stored.

```json
"LogData": {
   "FileNameMask": "goWebServer-%y-%m-%d.log",
   "Path": "somepath/logs",
   "MonitorSeconds": 30,
   "LogLevel": "verbose"
},
```

The 'Path' element will be prefixed with **ServerDataRoot** and is FIXED when the application loads. Environment values are substituted (Ref Environment Substitution").

The 'FileNameMask' has a separate substitution procedure implemented for the 'logging' module.

For example "goWebServer-%y-%m-%d-%H-%M-%S.log" will replace:

%y with a 4 character year

%m with a 2 character month

%d with a 2 character day of month

%H with a 2 character Hour (24 hour mode)

%M with a 2 character Minute

%S with a 2 character Second

After **MonitorSeconds** seconds the file name is generated again. If this is different to the current name the existing log is closed and a new log is opened with the new name. 

This process only occurs when the log is written to so it may be longer than the number of seconds defined in **MonitorSeconds** but it should never be less.

The log is written to AFTER the rename process so the latest log line will be appended to the latest log.

The 'LogLevel' element is not currently implemented.

## **TemplateStaticFiles**

```json
"TemplateStaticFiles": {
   "Files": [
      "dart.html",
      "dart.css"
   ],
   "DataFile": "configDataPI.json"
},
```

When a static file is read from the directory defined by **ServerStaticRoot** it is by default, returned without further processing. 

If the file name is included in the 'Files' list as above it is templated before it is returned.

This process allows Environment values to be embedded in the files before it is returned to the browser. 

Additional data is read from the global 'Env' element in the config document root and the file defined by the 'DataFile' element.

The file in the 'DataFile' element can be anywhare in the file systen. Its location is checked when the application loads. After that it is fixed.

# Environment Substitution

When the application loads the Operating System environment variables are read in to a cache. All values in the global 'Env' element are added to the cache. This may override OS environment variables.

When substitution takes place the environment variables ara always availiable.

Environment Substitution takes place when the application loads and fixes the main paths in the configuration file.

At the end of the load ALL paths should be absolute and fully defined. Where possible the files and paths are tested. If any fail the application aborts.

Runtime Environment Substitution also takes place under certain conditions. When this happens additional values can be defined using the User Env sections in the configuration file.

There are also some dynamically generated values for date and time.

## Substitution at load time

The values **ServerDataRoot** and **ServerStaticRoot** are substituted with only the OS Environment variables. The resulting paths are checked.

The value **LogData-->Path** is substituted with only the OS Environment variables. The resulting path are checked.

The value **faviconIcoPath** is substituted with only the OS Environment variables. The resulting file is checked.

### For Each User

For example:

```json
"stuart": {
   "name": "Stuart",
   "home": "stu"
   "Locations":{
      ...
   },
   "Env":{
      ...
   }
}
```

1. OS Environment variables
2. The 'name' (Stuart), 'home' (stu) and 'id' (stuart) user values are added from, for example, the stuart user shown above. 
3. The 'ms', 'doy', 'year', 'month', 'day', 'hour', 'min', and 'sec' values are added for the time of substitution. 
    'doy' = Day Of Year.
    'ms' = 'Unix Style' milliseconds since the epoch.
4. The Env name value pairs.

Each **Location** is substituted.

Each **Exec-->Log** is substituted with the OS Environment variables And User Environment variables. The resulting path are checked. 

Each **Exec-->Dir** is substituted with the OS Environment variables And User Environment variables. The resulting path are checked. 

Each **Exec-->Cmd** is substituted with the OS Environment variables And User Environment variables. 

Each **Exec-->LogOut** is substituted with the OS Environment variables And User Environment variables. 

Each **Exec-->LogErr** is substituted with the OS Environment variables And User Environment variables. 

### Static file Templating

When a static file is read from **ServerStaticRoot. For example via: 

```
http://localhost:8082/static/dart.css
```

If templating has been defined and loaded correctly the specified file is Substituted before returning to the browser. The OS Environment variables And the data from the **TemplateStaticFiles-->DataFile** are used.

### Exec command Templating

When an Exec definition is executed the command and all of its arguments are substituted first. The OS Environment variables And the data from the **TemplateStaticFiles-->DataFile** are used.

When the result of the exec command is returned and the configuration defines a **Exec-->LogOut** or **Exec-->LogErr** these are both templated. The OS Environment variables plus the additional time variables are used.

This allows to output to be stored using a time dependent file name.

## Substitution syntax

For ALL substitution except the log file names a simple markup is used.

```
%{name}
```

If the value for 'name' is not found then the markup remains unchanged.

Given that name='Stuart' the following substitutions will result:

```
My name is %{name}
My Name is Stuart

My name is %%{name}%
My Name is %Stuart%

My name is %%{Name}%
My Name is %%{Name}%
```
