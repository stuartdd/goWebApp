# goWebApp
Web app written in go for raspberry pi home server

Existring PI API

# Files

http://192.168.1.243:8080/files/user/stuart/loc/mydb
http://localhost:8082/files/user/stuart/loc/home
```json
{
   "user":"stuart",
   "loc":"mydb",
   "path":null,
   "files":[
      {
         "size":2515,
         "name":{
            "name":"testdata.data",
            "encName":"dGVzdGRhdGEuZGF0YQ=="
         }
      },
      {
         "size":320,
         "name":{
            "name":"test.json",
            "encName":"dGVzdC5qc29u"
         }
      },
      {
         "size":5812,
         "name":{
            "name":"stuff.json",
            "encName":"c3R1ZmYuanNvbg=="
         }
      }
   ]
}
```
# Files

http://192.168.1.243:8080/files/user/stuart/loc/thumbs/path/UGl4ZWxQaG9uZVN5bmMvSU1HXzIwMTcwNjAyXzEyNDExMw==
http://localhost:8082/files/user/stuart/loc/home/path/cy1waWNzL3MtdGVzdGZvbGRlcg==?a=b&x=y
```json
{
   "user":"stuart",
   "loc":"thumbs",
   "path":{
      "name":"PixelPhoneSync/IMG_20170602_124113",
      "encName":"UGl4ZWxQaG9uZVN5bmMvSU1HXzIwMTcwNjAyXzEyNDExMw=="
   },
   "files":[
      {
         "size":5917,
         "name":{
            "name":"2017_06_02_12_41_15_00002IMG_00002_BURST20170602124113.jpg.jpg",
            "encName":"MjAxN18wNl8wMl8xMl80MV8xNV8wMDAwMklNR18wMDAwMl9CVVJTVDIwMTcwNjAyMTI0MTEzLmpwZy5qcGc="
         }
      },
      {
         "size":5981,
         "name":{
            "name":"2017_06_02_12_41_15_00001IMG_00001_BURST20170602124113.jpg.jpg",
            "encName":"MjAxN18wNl8wMl8xMl80MV8xNV8wMDAwMUlNR18wMDAwMV9CVVJTVDIwMTcwNjAyMTI0MTEzLmpwZy5qcGc="
         }
      },
      {
         "size":5919,
         "name":{
            "name":"2017_06_02_12_41_14_00000IMG_00000_BURST20170602124113_COVER.jpg.jpg",
            "encName":"MjAxN18wNl8wMl8xMl80MV8xNF8wMDAwMElNR18wMDAwMF9CVVJTVDIwMTcwNjAyMTI0MTEzX0NPVkVSLmpwZy5qcGc="
         }
      }
   ]
}
```

# Paths

http://192.168.1.243:8080/paths/user/stuart/loc/thumbs
http://localhost:8082/paths/user/stuart/loc/home

```json
{
   "loc":"thumbs",
   "user":"stuart",
   "paths":[
      {
         "name":"LGPhone",
         "encName":"TEdQaG9uZQ=="
      },
      {
         "name":"PixelPhoneSync",
         "encName":"UGl4ZWxQaG9uZVN5bmM="
      },
      {
         "name":"PixelPhoneSync/IMG_20170602_124113",
         "encName":"UGl4ZWxQaG9uZVN5bmMvSU1HXzIwMTcwNjAyXzEyNDExMw=="
      },
      {
         "name":"PixelPhoneSync/IMG_20170810_160825",
         "encName":"UGl4ZWxQaG9uZVN5bmMvSU1HXzIwMTcwODEwXzE2MDgyNQ=="
      },
      {
         "name":"PixelPhoneSync/IMG_20180127_143024",
         "encName":"UGl4ZWxQaG9uZVN5bmMvSU1HXzIwMTgwMTI3XzE0MzAyNA=="
      }
   ]
}
```