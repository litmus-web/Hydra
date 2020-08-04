import aiohttp
import asyncio


async def main():
    async with aiohttp.ClientSession() as sess:
        ws = await sess.ws_connect("ws://127.0.0.1:8000/workers")
        await ws.send_str("Hello world")
        msg = await ws.receive_str()
        print(msg)
        await ws.send_str("Hello world")



if __name__ == '__main__':
    asyncio.run(main())