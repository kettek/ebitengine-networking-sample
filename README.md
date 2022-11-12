This is a _very_ simple server + client networking example using TCP and the gob encoder. It is kept as minimal as possible in terms of abstract data structures.

After building, the host must start the server via:
```
./ebitengine-networking-sample host 127.0.0.1:9999
```

After which, the client can connect via:
```
./ebitengine-networking-sample join 127.0.0.1:9999
```

Once connected, each "player"'s ebiten can be controlled with the arrow keys.

This is not intended as an instructional for how to organize networking code and logic, but rather a barebones example of how to actually get basic communication working.
