from flask import Flask, session
from workers import WsgiAdapter

app = Flask(__name__)
app.secret_key = "gagagasdgfasdgadfg"

@app.route("/")
def hello_world():
    return "hello world"


if __name__ == '__main__':
    wsgi = WsgiAdapter(app)
    wsgi.start()