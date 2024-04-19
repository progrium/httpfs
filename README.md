# httpfs

Create filesystems using your favorite HTTP framework.


```
$ tree ~/demo
/Users/progrium/demo
├── greet
│   ├── goodbye
│   └── hello
└── random

2 directories, 3 files
```

That file tree mounted at `~/demo` is powered by this Flask app:

```python
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
```

All it took was one command:
```
httpfs -mount ~/demo ./examples/flask-basic/app.py
```

## Install

Currently works on Linux and Mac (with [MacFUSE](https://osxfuse.github.io/)).

```
go install github.com/progrium/httpfs
```

## Build

Check out the [examples directory](examples) or read the [PROTOCOL.md](PROTOCOL.md) to see how it works.

## License

MIT
