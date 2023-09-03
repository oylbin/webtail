# webtail.exe python foo.py
# python bar.py

import websockets
import asyncio

async def run():
    uri = "ws://localhost:17862/logs"
    async with websockets.connect(uri) as ws:
        async for message in ws:
            print(message)

asyncio.run(run())