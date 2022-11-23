# gorrent

## Features
- ~~Only single file torrent is supported~~
- Do not support upload
- Do not support DHT

## How it Works
1. Find peers
   1. parse a .torrent file
   2. retrieve peers from the tracker
2. Download from peers
   1. start a TCP connection
   2. complete BitTorrent peer protocol handshake
   3. send & receive messages
      + interpreting different types of messages 
      + manage concurrency & state
      + pipelining requests
   4. put it all together

## References
+ https://blog.jse.li/posts/torrent
+ https://github.com/archeryue/go-torrent
+ https://www.bittorrent.org/beps/bep_0003.html
+ https://wiki.theory.org/BitTorrentSpecification