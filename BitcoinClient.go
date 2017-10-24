package doriath

// BitcoinClient is an interface to a Bitcoin client providing the functionality needed by doriath. A sample implementation wrapping Bitcoin Core's JSON-RPC is provided as BitcoinCoreClient. All hashes are in "correct" order rather than Bitcoin's customarily reversed printing order.
type BitcoinClient interface {
	// Obtains the total number of blocks in the canonical chain
	GetBlockCount() (int, error)
	// Obtains a particular block header by block index
	GetHeader(idx int) ([]byte, error)
	// LocateTx returns the index of the block which contains the transaction identified by hash
	LocateTx(txhsh []byte) (int, error)
	// GetMerkle returns the merkle branch to a confirmed transaction given its hash
	GetMerkle(txhsh []byte, blkidx int) (mk [][]byte, pos int, err error)
	// EstimateFee returns recommended fee, in satoshis per byte
	EstimateFee(within int) float64
	// Signs a transaction in binary form, given a private key in WIF format
	SignTx(tx []byte, skWIF string) ([]byte, error)
	// Broadcasts a transaction to the Bitcoin network
	BroadcastTx(tx []byte) error
}
