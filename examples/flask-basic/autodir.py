from flask import request, jsonify, Response

# AutoDirMiddleware reflects on routes and makes handlers for parent
# directories of route paths that serve an httpfs directory
class AutoDirMiddleware:
    def __init__(self, app):
        self.app = app
        self.app.wsgi_app = self.middleware(self.app.wsgi_app)
    
    def middleware(self, next_app):
        def _middleware(environ, start_response):
            with self.app.request_context(environ):
                full_paths = {str(rule) for rule in self.app.url_map.iter_rules()}
                if request.path in full_paths:
                    return next_app(environ, start_response)
                
                subpaths = self.get_subpaths(request.path, full_paths)
                if subpaths:
                    response = jsonify({"dir": list(subpaths)})
                    response.headers['Content-Type'] = 'application/vnd.httpfs.v1+json'
                    return response(environ, start_response)
                
                response = Response("Not found", status=404)
                return response(environ, start_response)
                
        return _middleware

    def get_subpaths(self, path, all_paths):
        if path != '/':
          path = '/' + path.strip('/') + '/'
        subpaths = []
        for p in all_paths:
            if p.startswith("/static"):
                continue
            if p.startswith(path):
                rest = p[len(path):].split("/")
                if rest[0].startswith("<"):
                    continue
                if len(rest) == 1:
                  subpaths.append(rest[0])
                else:
                  subpaths.append(rest[0]+'/')
        return set(subpaths)
