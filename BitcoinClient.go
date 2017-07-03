package doriath

// BitcoinClient is an interface to a Bitcoin client providing the functionality needed by doriath. A sample implementation wrapping Bitcoin Core's JSON-RPC is provided as BitcoinCoreClient. All hashes are in "correct" order rather than Bitcoin's customarily reversed printing order.
type BitcoinClient interface {
	// Obtains the total number of blocks in the canonical chain
	GetBlockCount() (int, error)
	// Obtains the block index given a block hash
	//GetBlockIdx([idx int]) (int, error)
	// Obtains a particular block by block index
	GetBlock(idx int) ([]byte, error)
	// Obtains a particular block header by block index
	GetHeader(idx int) ([]byte, error)
	// LocateTx returns the index of the block which contains the transaction identified by hash
	LocateTx(txhsh []byte) (int, error)
	// Signs a transaction in binary form, given a private key in WIF format
	SignTx(tx []byte, skWIF string) ([]byte, error)
	// Broadcasts a transaction to the Bitcoin network
	BroadcastTx(tx []byte) error
}
