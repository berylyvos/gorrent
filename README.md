# gorrent

### How it Works
1. Find peers
   1. parse a .torrent file
   2. retrieve peers from the tracker
2. Download from peers
   1. start a TCP connection & handshake
   2. send & receive messages
      1. interpret messages
   3. put it all together
      1. manage concurrency & state
      2. pipeline requests 

### References
+ https://blog.jse.li/posts/torrent/
+ https://github.com/archeryue/go-torrent