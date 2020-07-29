import uvicorn

async def app(scope, receive, send):
    assert scope['type'] == 'http'
    await send({
        'type': 'http.response.start',
        'status': 200,
        'headers': [
            [b'content-type', b'text/html'],
        ]
    })
    await send({
        'type': 'http.response.body',
        'body': b"Hello world",
    })


if __name__ == "__main__":
    uvicorn.run("uvicorn_server:app", host="0.0.0.0", port=5050, log_level="info", access_log=False)
