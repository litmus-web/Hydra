# Using Hydra Webserver
Welcome to the Hydra usage docs, this will show you how to setup Hydra to run on your system.

## A Basic Guide
### 1) Installing Dependancies
- You can install Hydra using `pip install hydra-client` **this installs the client only for Python**, 
- You will need to compile the main server (the `hydra` folder) to a executable from source or download one of our read made binaries.
- **Optional Speedups** - To improve Hydra's performance further you can pip install the following: `uvloop`, `aiohttp[speedups]`

### 2) Setting up your project
In your target python file you should add a few things to allow Hydra to interact with it.

**Example of a target file with the ASGI adapter**
```py
# ./my_file.py

from hydra_client import run

async def app(scope, receive, send):
    assert scope['type'] == 'http'
    await send({
        'type': 'http.response.start',
        'status': 200,
        'headers': [
            [b'content-type', b'text/plain'],
        ]
    })
    await send({
        'type': 'http.response.body',
        'body': b'Hello, world!',
    })

if __name__ == "__main__":
  run()
```  

### 3) Running with CLI
After we've added the nessesary code to our files we can run the server using:


`hydra --app "my_file:app" --adapter "asgi" --host "0.0.0.0:5050"`

Hey presto! We now have a running server!


## Options And Configuration

 **Required**
- `--app` - The target file and app callable seperated by a `:`, e.g. `my_file:app`
- `--adapter` - The adapter type, this can be `asgi`, `wsgi` or `raw` depending on your framework 

 **General**
- `--host` - The binding host address and port, e.g. `0.0.0.0:5050`<br>
        **Default:** `127.0.0.1:8080`<br>
        
- `--workers` - The amount of workers to spawn, the amount of processes spawned is equal to `2 * workers + 1`<br>
        **Recommeneded:** `2 * num_threads`<br>
        **Default:** `1` worker<br>

**Low Level Control**
- `--shardsperproc` - Set the amount of WS connections (Shards) to connect to Hydra, you should only use this if you are doing very specific load balancing.

- `--name` - Sets the server name.

- `--maxconnperip` - Sets the maximum number of client connections per IP.

- `--maxreqperconn` - The maximum number of requests allowed per connection.

- `--tcpkeepalive` - Enable/Disable TCP keep alive.

- `--reducememory` - Start the server in reduce memory mode, this will try to minimuse the amount of memory used. (May effect performance)

- `--maxreqsize` - Specify the maximum body size allowed in a request, useful if you want to protect your server from attacks.
