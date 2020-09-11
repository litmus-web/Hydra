# Using Hydra Webserver
Welcome to the Hydra usage docs, this will show you how to setup Hydra to run on your system.

### 1) Installing Dependancies
- You can install Hydra using `pip install hydra-client` **this isntalls the client only for Python**, 
- You will need to compile the main server (the `hydra` folder) to a executable from source or download one of our read made binaries.
- **Optional Speedups** - To improve Hydra's performance further you can pip install the following: `uvloop`, `aiohttp[speedups]`

### 2) Setting up your project
In your target python file you should add a few things to allow Hydra to interact with it.

**Example of a target file with the RAW adapter**
```py
from hydra_client import run, RawRequest

async def my_app(req: RawRequest):
  pass

if __name__ == "__main__":
  run()
```  
