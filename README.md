# gorrent

## Features
- Multiple files torrents download
- Supports UDP & HTTP trackers
- ~~Supports uploading pieces~~
- ~~Supports DHT, PeX and Magnet links~~

## How it Works
1. Peers discovery
   1. parse a .torrent file
   2. retrieve peers info(i.e. IP, port) from UDP/HTTP trackers
2. Download from peers
   1. start a TCP connection
   2. complete BitTorrent peer protocol handshake
   3. exchange messages
      + interpreting different types of messages 
      + manage concurrency & state
      + pipelining requests
   4. assemble data

## References
+ [BEP-3: The BitTorrent Protocol Specification](https://www.bittorrent.org/beps/bep_0003.html)
+ [Bittorrent Protocol Specification v1.0](https://wiki.theory.org/BitTorrentSpecification)
+ [A toy torrent client written in golang](https://github.com/archeryue/go-torrent)
+ [Building a BitTorrent client from the ground up in Go](https://blog.jse.li/posts/torrent)
+ [BEP-15: UDP Tracker Protocol for BitTorrent](http://bittorrent.org/beps/bep_0015.html)
+ [BitTorrent’s Future: DHT, PEX, and Magnet Links Explained](https://lifehacker.com/bittorrent-s-future-dht-pex-and-magnet-links-explain-5411311)