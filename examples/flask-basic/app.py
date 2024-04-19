#!/usr/bin/env python3

import os, random
from flask import Flask
from autodir import AutoDirMiddleware

app = Flask(__name__)
AutoDirMiddleware(app)

@app.get("/greet/hello")
def hello():
    return "Hello, world!\n"

@app.get("/greet/goodbye")
def goodbye():
    return "Goodbye, world!\n"

@app.get("/random")
def rnd():
    return ''.join(random.choice('abcdefghijklmnopqrstuvwyz') for _ in range(24))+"\n"


if __name__ == "__main__":
    port = int(os.getenv('PORT', 5000))
    app.run(debug=True, port=port)
