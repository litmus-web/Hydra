from quart import Quart
from workers import ASGIAdapter

app = Quart(__name__)

@app.route("/")
async def index():
    return "hello world"

if __name__ == "__main__":
    asgi = ASGIAdapter(app)
    asgi.start()
