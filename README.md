# inet-peercred
A simple server to provide the peer credentials of an inet socket to a remote host. Same idea as `SO_PEERCRED` for unix sockets. Runs an https server on port 411. 
The only security is that it requires the client to be connecting from a privileged port (<=1024).
## Endpoints
### /v1/query
Expects the body to contain the query, like:
```
{
        "local_addr": {
                "ip": "10.31.104.204",
                "port": 45532
        },
        "remote_addr": {
                "ip": "10.32.108.191",
                "port": 9998
        }
}
```
local/remote is from the perspective of the server. Only full matches are supported.
Result:
```
{
	"user" {
		"real":"root",
		"effective":"root",
		"saved":"root",
		"filesystem":"root"
		},
	"groups":{
		"real":"root",
		"effective":"root",
		"saved":"root",
		"filesystem":"root"
		},
	"supplementary_groups":null
}
```
