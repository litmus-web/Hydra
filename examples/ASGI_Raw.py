from workers import ASGIAdapter

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
        'body': "hello world".encode('utf-8'),
    })


if __name__ == '__main__':
    asgi = ASGIAdapter(app)
    asgi.start()
