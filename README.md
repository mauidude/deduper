# Dedeuper

An HTTP server that allows you to find near duplicate or similar documents given another document.
Implements go-raft so it can run as a cluster with other nodes and provide high-availability.

## Installation

```sh
go get github.com/mauidude/deduper
```

## Building

```sh
godep go build
```

## Running

```sh
./deduper [data directory]
```

### Options

- `-host` The host the server will run on. Defaults to `localhost`.
- `-port` The port the server will run on. Defaults to `8080`.
- `-leader` The `host:port` of the leader node, if running as a follower. Defaults to leader mode.
- `-debug` Enables debug output. Defaults to `false`.

The following options will require testing with your document sizes and overall corpus size.
**If you change these values, you will need to readd all of your documents.**

- `-bands` The number of bands to use in the minhash algorithm. Defaults to `100`.
- `-hashes` The number of hashes to use in the minhash algorithm. Defaults to `2`.
- `-shingles` The shingle size to use on the text. Defaults to `2`.

## Testing

```sh
godep go test ./...
```

## API

### Adding a document

```
POST /documents/:id HTTP/1.1
[HTTP headers...]

[document body]
```

This will add the document to the index under the given `id`.

Writes can be given to a leader or follower. Any writes to a follower get
proxied to the leader.

### Finding similar documents

```
POST /documents/similar HTTP/1.1
[HTTP headers...]

[document body]
```

This `POST` takes an optional `threshold` argument in the query string which will return only
documents with a similarity greater than or equal to that value. This value must be between
`0` and `1`. The default is `0.8`.

This will return a JSON object of matching documents and their similarity. Similarity is a
value between `0` and `1` where `1` is identical and `0` is no shared content.

```json
[
    {
        "id": "mydocument.txt",
        "similarity": 0.934
    },
    {
        "id": "someotherdocument.txt",
        "similarity": 0.85
    }
]
```
