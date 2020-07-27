import uvloop

from workers import ASGIAdapter

uvloop.install()

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
        'body': b"hello world",
    })


if __name__ == '__main__':
    asgi = ASGIAdapter(app)
    asgi.start()
