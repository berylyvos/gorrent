module github.com/berylyvos/gorrent/torrent

go 1.19

require (
	github.com/berylyvos/gorrent/bencode v0.0.0-20221105170631-94cf0abec1cd
	github.com/stretchr/testify v1.8.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/berylyvos/gorrent/bencode => ../bencode
)