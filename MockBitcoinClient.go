package doriath

import (
	"bytes"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

// MockBitcoinClient is a client to a fake, local, sorta-valid Bitcoin blockchain, with blocks being mined every minute with no proof-of-work and no signature checking.
type MockBitcoinClient struct {
	blocks   []libkataware.Block
	blockIdx map[string]int
	txIdx    map[string]int
	mempool  []libkataware.Transaction

	lk sync.RWMutex
}

// GetBlockCount obtains the total number of blocks in the fake blockchain.
func (mbc *MockBitcoinClient) GetBlockCount() (bcount int, err error) {
	mbc.lk.RLock()
	defer mbc.lk.RUnlock()
	return len(mbc.blocks), nil
}

// GetBlockHash converts an index to a block hash in the fake blockchain.
func (mbc *MockBitcoinClient) GetBlockHash(idx int) (hsh []byte, err error) {
	mbc.lk.RLock()
	defer mbc.lk.RUnlock()
	return libkataware.DoubleSHA256(mbc.blocks[idx].Hdr.Serialize()), nil
}

// GetBlockIdx converts a block hash to a block index.
func (mbc *MockBitcoinClient) GetBlockIdx(hsh []byte) (idx int, err error) {
	mbc.lk.RLock()
	defer mbc.lk.RUnlock()
	idx, ok := mbc.blockIdx[string(hsh)]
	if !ok {
		err = errors.New("not found")
	}
	return
}

// GetBlock gets a block by its hash.
func (mbc *MockBitcoinClient) GetBlock(idx int) ([]byte, error) {
	mbc.lk.RLock()
	defer mbc.lk.RUnlock()
	return mbc.blocks[idx].Serialize(), nil
}

// GetHeader gets a block header by the index.
func (mbc *MockBitcoinClient) GetHeader(idx int) ([]byte, error) {
	blk, err := mbc.GetBlock(idx)
	if err != nil {
		return nil, err
	}
	return blk[:80], nil
}

// LocateTx returns the hash of the block that contains a certain transaction.
func (mbc *MockBitcoinClient) LocateTx(txhsh []byte) (int, error) {
	mbc.lk.RLock()
	defer mbc.lk.RUnlock()
	bhsh, ok := mbc.txIdx[string(txhsh)]
	if !ok {
		return -1, errors.New("no such tx")
	}
	return bhsh, nil
}

func (mbc *MockBitcoinClient) garbageTx() []byte {
	gbg := make([]byte, 1024)
	rand.Read(gbg)
	badtx := libkataware.Transaction{
		Version: 1,
		Inputs: []libkataware.TxInput{
			libkataware.TxInput{
				PrevHash: make([]byte, 32),
				Script:   gbg,
			},
		},
	}
	return badtx.ToBytes()
}

// SignTx "signs" a transaction; in this case it's a no-op.
func (mbc *MockBitcoinClient) SignTx(tx []byte, skWIF string) ([]byte, error) {
	return tx, nil
}

// BroadcastTx broadcasts a transaction to the fake blockchain.
func (mbc *MockBitcoinClient) BroadcastTx(tx []byte) error {
	mbc.lk.RLock()
	defer mbc.lk.RUnlock()
	var lktx libkataware.Transaction
	err := lktx.Unpack(bytes.NewReader(tx))
	if err != nil {
		return err
	}
	mbc.mempool = append(mbc.mempool, lktx)
	return nil
}

// NewMockBitcoinClient creates a new mock Bitcoin client, together with a bogus funds source.
func NewMockBitcoinClient() (*MockBitcoinClient, []byte) {
	toret := &MockBitcoinClient{
		blockIdx: make(map[string]int),
		txIdx:    make(map[string]int),
	}
	go func() {
		for {
			time.Sleep(time.Second * 10)
			for i := 0; i < rand.Int()%50+700; i++ {
				e := toret.BroadcastTx(toret.garbageTx())
				if e != nil {
					panic(e)
				}
			}
			toret.lk.Lock()
			oldbhash := make([]byte, 32)
			if len(toret.blocks) != 0 {
				oldbhash = libkataware.DoubleSHA256(toret.blocks[len(toret.blocks)-1].Hdr.Serialize())
			}
			var newblk libkataware.Block
			newblk.Hdr.HashPrevBlock = oldbhash
			newblk.Hdr.Nonce = rand.Uint32()
			newblk.Hdr.Time = time.Now()
			newblk.Hdr.Version = 4
			newblk.Bdy = toret.mempool
			toret.mempool = nil
			if len(newblk.Bdy) != 0 {
				mkl, pos := newblk.GenMerkle(newblk.Bdy[0].Hash256())
				newblk.Hdr.HashMerkleRoot = newblk.Hdr.FixedMerkleRoot(mkl, pos, newblk.Bdy[0])
			}
			toret.blocks = append(toret.blocks, newblk)
			toret.blockIdx[string(libkataware.DoubleSHA256(newblk.Hdr.Serialize()))] =
				len(toret.blocks) - 1
			//blkhsh := libkataware.DoubleSHA256(newblk.Hdr.Serialize())
			for _, tx := range newblk.Bdy {
				toret.txIdx[string(tx.Hash256())] = len(toret.blocks) - 1
			}
			toret.lk.Unlock()
		}
	}()
	bogusTx := libkataware.Transaction{
		Version: 1,
		Inputs: []libkataware.TxInput{
			libkataware.TxInput{PrevHash: make([]byte, 32)},
		},
		Outputs: []libkataware.TxOutput{
			libkataware.TxOutput{
				Value: 1000000,
			},
		},
	}
	return toret, bogusTx.ToBytes()
}
