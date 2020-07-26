from flask import Flask, session
from workers import WSGIAdapter

app = Flask(__name__)
app.secret_key = "gagagasdgfasdgadfg"

@app.route("/bob")
def hello_world():
    return "hello world"


if __name__ == '__main__':
    wsgi = WSGIAdapter(app)
    wsgi.start()