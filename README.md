# gorrent

### How it Works
1. Find peers
   1. parse a .torrent file
   2. retrieve peers from the tracker
2. Download from peers
   1. start a TCP connection
   2. complete BitTorrent peer protocol handshake
   3. send & receive messages
      + interpret messages
   4. put it all together
      + manage concurrency & state
      + pipelining requests 

### References
+ https://blog.jse.li/posts/torrent
+ https://github.com/archeryue/go-torrent
+ https://www.bittorrent.org/beps/bep_0003.html