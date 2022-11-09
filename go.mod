module github.com/berylyvos/gorrent

go 1.19

replace (
	github.com/berylyvos/gorrent/bencode => ./bencode
	github.com/berylyvos/gorrent/torrent => ./torrent
)

require github.com/berylyvos/gorrent/torrent v0.0.0-20221109050236-e6280721ec09

require github.com/berylyvos/gorrent/bencode v0.0.0-20221105170631-94cf0abec1cd // indirect
