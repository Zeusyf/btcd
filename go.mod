module github.com/zeusyf/btcd

go 1.22.1

replace github.com/zeusyf/btcd/blockchain => ./blockchain

replace github.com/zeusyf/btcd/blockchain/chainutil => ./blockchain/chainutil

replace github.com/zeusyf/btcd/blockchain/indexers => ./blockchain/indexers
replace github.com/zeusyf/btcd/chaincfg => ./chaincfg

replace github.com/zeusyf/btcd/chaincfg/chainhash => ./chaincfg/chainhash

replace github.com/zeusyf/btcd/database => ./database

replace github.com/zeusyf/btcd/database/ffldb => ./database/ffldb

replace github.com/zeusyf/btcd/wire => ./wire

replace github.com/zeusyf/btcd/wire/common => ./wire/common

replace github.com/zeusyf/btcd/btcjson => ./btcjson

replace github.com/zeusyf/btcd/limits => ./limits

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/jessevdk/go-flags v1.6.1
	github.com/zeusyf/btcd/blockchain v0.0.0
	github.com/zeusyf/btcd/blockchain/chainutil v0.0.0
	github.com/zeusyf/btcd/blockchain/indexers v0.0.0
	github.com/zeusyf/btcd/btcjson v0.0.0
	github.com/zeusyf/btcd/chaincfg v0.0.0
	github.com/zeusyf/btcd/chaincfg/chainhash v0.0.0
	github.com/zeusyf/btcd/database v0.0.0
	github.com/zeusyf/btcd/database/ffldb v0.0.0
	github.com/zeusyf/btcd/limits v0.0.0
	github.com/zeusyf/btcd/wire v0.0.0
	github.com/zeusyf/btcd/wire/common v0.0.0
	github.com/zeusyf/btclog v0.0.0-20250116184953-8c8a19140fb8
	github.com/zeusyf/btcutil v0.0.0-20250116184958-9199c311bd5f
	github.com/zeusyf/go-socks/socks v0.0.0-20250116185002-7366ac3abc4b
	github.com/zeusyf/omega/ovm v0.0.0-20250116185010-5e096d407f09
	github.com/zeusyf/omega/token v0.0.0-20250116185010-5e096d407f09
	github.com/zeusyf/websocket v0.0.0-20250116185018-c83814635629
)

require (
	github.com/aead/siphash v1.0.1 // indirect
	github.com/goinggo/mapstructure v0.0.0-20140717182941-194205d9b4a9 // indirect
	github.com/kkdai/bstream v1.0.0 // indirect
	github.com/zeusyf/btcd/btcec v0.0.0-00010101000000-000000000000 // indirect
	github.com/zeusyf/goleveldb/leveldb v0.0.0-20250116185006-f5643458dbf2 // indirect
	github.com/zeusyf/goleveldb/leveldb/comparer v0.0.0-20250116185006-f5643458dbf2 // indirect
	github.com/zeusyf/goleveldb/leveldb/errors v0.0.0-20250116185006-f5643458dbf2 // indirect
	github.com/zeusyf/goleveldb/leveldb/filter v0.0.0-20250116185006-f5643458dbf2 // indirect
	github.com/zeusyf/goleveldb/leveldb/iterator v0.0.0-20250116185006-f5643458dbf2 // indirect
	github.com/zeusyf/goleveldb/leveldb/opt v0.0.0-20250116185006-f5643458dbf2 // indirect
	github.com/zeusyf/goleveldb/leveldb/util v0.0.0-20250116185006-f5643458dbf2 // indirect
	github.com/zeusyf/omega v0.0.0-20250116185010-5e096d407f09 // indirect
	github.com/zeusyf/omega/viewpoint v0.0.0-20250116185010-5e096d407f09 // indirect
	github.com/zeusyf/snappy-go v0.0.0-20250116185016-0a1472be23f6 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
)
