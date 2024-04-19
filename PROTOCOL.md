# httpfs protocol

This document describes how httpfs uses HTTP to produce filesystem data.

## Connection

httpfs starts your HTTP server from the command argument you pass it and
sets `PORT` with an unused port in its environment for you to listen on.

## Methods

In this read-only iteration, httpfs only uses `GET` and `HEAD` methods.

`HEAD` is used for getting file metadata and is correlated with `stat`
operations. Since systems often perform `stat` quite a bit in a row, httpfs
caches HEAD requests for 1 second regardless of cache headers.

`GET` is used for getting file data and is correlated with `open` operations.
Streaming data is not supported.

## Status Codes

httpfs only understands `200` and `404` response status codes and anything else
will result in the `EINVAL` (invalid argument) error code being used.

## Response Headers

httpfs uses the following response headers:

#### Content-Length

Used for size metadata. If not present, a size of `0` will be used.

#### Last-Modified

Used for modified time (mtime) metadata. If not present, httpfs will use its
start time.

#### Content-Permissions

This non-standard header is used for permission metadata. It is expected to be
in octal format prefixed with a `0` (example: `0755`). If not present, httpfs
will use `0644` for files and `0755` for directories.

#### Content-Disposition

Used to specify a filename different from the path using the `filename`
attribute of the `attachment` disposition (example: 
`attachment; filename="filename.jpg"`). If not present, the filename will be 
the basename of the URL path.

#### Content-Type

This is ignored for files, but for directories it is expected to be
`application/vnd.httpfs.v1+json`.

## Directories

Directory URLs need to serve JSON data with Content-Type
`application/vnd.httpfs.v1+json`. The response body JSON must be an object with
a `dirs` property containing an array of strings for directory contents. The
strings are expected to be the basename of files and the basename of directories
with a `/` suffix to identify them as directories. Here is an example response
body:

```json
{
  "dirs": [
    "dirA/",
    "dirB/",
    "file1.txt",
    "file2.txt"
  ]
}
```

Since most HTTP frameworks let you specify handlers to full paths/routes, you
are responsible for making sure the parent directory paths are served as
directories, **including the root.** This can be accomplished automatically with
middleware in most cases. However, if you have routes with dynamic/variable path
elements, these cannot be automatically served and you will have to make a
directory handler to enumerate all directory entries.
