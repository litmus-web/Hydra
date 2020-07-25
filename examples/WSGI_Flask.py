from flask import Flask
from workers import WsgiAdapter

app = Flask(__name__)

@app.route("/")
def hello_world():
    return "hello world"


if __name__ == '__main__':
    wsgi = WsgiAdapter(app)
    wsgi.start()