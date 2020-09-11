# Using Hydra Webserver
Welcome to the Hydra usage docs, this will show you how to setup Hydra to run on your system.

## A Basic GGuide
### 1) Installing Dependancies
- You can install Hydra using `pip install hydra-client` **this isntalls the client only for Python**, 
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

After we've added the nessesary code to our files we can run the server!

**Running the server**
`hydra --app "my_file:app" --adapter "asgi" --host "0.0.0.0:5050"`

Hey presto! We now have a running server!

