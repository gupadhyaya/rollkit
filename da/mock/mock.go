package mock

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/celestiaorg/rollmint/da"
	"github.com/celestiaorg/rollmint/log"
	"github.com/celestiaorg/rollmint/store"
	"github.com/celestiaorg/rollmint/types"
	ds "github.com/ipfs/go-datastore"
)

// DataAvailabilityLayerClient is intended only for usage in tests.
// It does actually ensures DA - it stores data in-memory.
type DataAvailabilityLayerClient struct {
	logger   log.Logger
	dalcKV   ds.Datastore
	daHeight uint64
	config   config
}

const defaultBlockTime = 3 * time.Second

type config struct {
	BlockTime time.Duration
}

var _ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
var _ da.BlockRetriever = &DataAvailabilityLayerClient{}

// Init is called once to allow DA client to read configuration and initialize resources.
func (m *DataAvailabilityLayerClient) Init(_ types.NamespaceID, config []byte, dalcKV ds.Datastore, logger log.Logger) error {
	m.logger = logger
	m.dalcKV = dalcKV
	m.daHeight = 1
	if len(config) > 0 {
		var err error
		m.config.BlockTime, err = time.ParseDuration(string(config))
		if err != nil {
			return err
		}
	} else {
		m.config.BlockTime = defaultBlockTime
	}
	return nil
}

// Start implements DataAvailabilityLayerClient interface.
func (m *DataAvailabilityLayerClient) Start() error {
	m.logger.Debug("Mock Data Availability Layer Client starting")
	go func() {
		for {
			time.Sleep(m.config.BlockTime)
			m.updateDAHeight()
		}
	}()
	return nil
}

// Stop implements DataAvailabilityLayerClient interface.
func (m *DataAvailabilityLayerClient) Stop() error {
	m.logger.Debug("Mock Data Availability Layer Client stopped")
	return nil
}

// SubmitBlock submits the passed in block to the DA layer.
// This should create a transaction which (potentially)
// triggers a state transition in the DA layer.
func (m *DataAvailabilityLayerClient) SubmitBlock(ctx context.Context, block *types.Block) da.ResultSubmitBlock {
	daHeight := atomic.LoadUint64(&m.daHeight)
	m.logger.Debug("Submitting block to DA layer!", "height", block.Header.Height, "dataLayerHeight", daHeight)

	hash := block.Header.Hash()
	blob, err := block.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBlock{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}

	err = m.dalcKV.Put(ctx, getKey(daHeight, block.Header.Height), hash[:])
	if err != nil {
		return da.ResultSubmitBlock{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}

	err = m.dalcKV.Put(ctx, ds.NewKey(hex.EncodeToString(hash[:])), blob)
	if err != nil {
		return da.ResultSubmitBlock{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}

	return da.ResultSubmitBlock{
		BaseResult: da.BaseResult{
			Code:     da.StatusSuccess,
			Message:  "OK",
			DAHeight: daHeight,
		},
	}
}

// CheckBlockAvailability queries DA layer to check data availability of block corresponding to given header.
func (m *DataAvailabilityLayerClient) CheckBlockAvailability(ctx context.Context, daHeight uint64) da.ResultCheckBlock {
	blocksRes := m.RetrieveBlocks(ctx, daHeight)
	return da.ResultCheckBlock{BaseResult: da.BaseResult{Code: blocksRes.Code}, DataAvailable: len(blocksRes.Blocks) > 0}
}

// RetrieveBlocks returns block at given height from data availability layer.
func (m *DataAvailabilityLayerClient) RetrieveBlocks(ctx context.Context, daHeight uint64) da.ResultRetrieveBlocks {
	if daHeight >= atomic.LoadUint64(&m.daHeight) {
		return da.ResultRetrieveBlocks{BaseResult: da.BaseResult{Code: da.StatusError, Message: "block not found"}}
	}

	entries, err := store.PrefixEntries(ctx, m.dalcKV, getPrefix(daHeight))
	if err != nil {
		return da.ResultRetrieveBlocks{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}

	var blocks []*types.Block
	for _, entry := range entries {
		hash, err := m.dalcKV.Get(ctx, ds.NewKey(entry.Key))
		if err != nil {
			return da.ResultRetrieveBlocks{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
		}

		blob, err := m.dalcKV.Get(ctx, ds.NewKey(hex.EncodeToString(hash)))
		if err != nil {
			return da.ResultRetrieveBlocks{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
		}

		block := &types.Block{}
		err = block.UnmarshalBinary(blob)
		if err != nil {
			return da.ResultRetrieveBlocks{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
		}
		blocks = append(blocks, block)
	}

	return da.ResultRetrieveBlocks{BaseResult: da.BaseResult{Code: da.StatusSuccess}, Blocks: blocks}
}

func getPrefix(daHeight uint64) string {
	var buf bytes.Buffer
	buf.WriteString("/")
	buf.WriteString(strconv.FormatUint(daHeight, 10))
	return buf.String() // returns `/daHeight`
}

func getKey(daHeight uint64, height uint64) ds.Key {
	var buf bytes.Buffer
	buf.WriteString(strconv.FormatUint(daHeight, 10))
	buf.WriteString("/")
	buf.WriteString(strconv.FormatUint(height, 10))
	return ds.NewKey(buf.String()) // returns `/daHeight/height`
}

func (m *DataAvailabilityLayerClient) updateDAHeight() {
	blockStep := rand.Uint64()%10 + 1
	atomic.AddUint64(&m.daHeight, blockStep)
}
