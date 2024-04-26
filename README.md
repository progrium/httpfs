# httpfs

Create virtual filesystems using any HTTP framework.


```
$ tree ~/demo
/Users/progrium/demo
├── greet
│   ├── goodbye
│   └── hello
└── random

2 directories, 3 files
```

The file tree mounted at `~/demo` is powered by this Flask app:

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

Don't care for Flask? Use any web framework!

## Install

Currently works on Linux and Mac (with [MacFUSE](https://osxfuse.github.io/)). Download from [latest release](https://github.com/progrium/httpfs/releases/latest) or you can run this installer:

```sh
bash -c "$(curl -sSL https://raw.githubusercontent.com/progrium/httpfs/main/install.sh)"
```

Alternatively you can install using [Homebrew](https://brew.sh/):

```sh
brew tap progrium/homebrew-taps
brew install httpfs
```

## Build an HTTP filesystem

Check out the [examples directory](examples) or read the [PROTOCOL.md](PROTOCOL.md) to see how it works.

## License

MIT
