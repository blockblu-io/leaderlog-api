package blockfrost

import (
	"context"
	"fmt"
	"github.com/blockblu-io/leaderlog-api/pkg/chain"
	"github.com/blockfrost/blockfrost-go"
	"os"
)

// Backend is an implementation of chain.Backend that makes use of the
// Blockfrost API.
type Backend struct {
	client blockfrost.APIClient
	cache  poolCache
}

// NewBlockFrostBackend is creating a new chain.Backend that uses Blockfrost API
// with the api key specified in the environment.
func NewBlockFrostBackend() (chain.Backend, error) {
	apiKey := os.Getenv("BLU_BLOCKFROST_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API key for blockfrost hasn't been specified in the environment (BLU_BLOCKFROST_API_KEY)")
	}
	client := blockfrost.NewAPIClient(blockfrost.APIClientOptions{
		ProjectID: apiKey,
	})
	return &Backend{
		client: client,
		cache:  newPoolCache(5),
	}, nil
}

func (b *Backend) Name() string {
	return "blockfrost"
}

func (b *Backend) GetLatestBlock(ctx context.Context) (*chain.Tip, error) {
	block, err := b.client.BlockLatest(ctx)
	if err != nil {
		return nil, err
	}
	return &chain.Tip{
		Height:      uint(block.Height),
		Hash:        block.Hash,
		Epoch:       uint(block.Epoch),
		SlotInEpoch: uint(block.EpochSlot),
		Slot:        uint(block.Slot),
		Timestamp:   uint(block.Time),
	}, nil
}

// fetchPoolMetadata returns a function that can be used with the pool
// cache to fetch metadata of a pool.
func (b *Backend) fetchPoolMetadata() fetchPoolMetadataFunc {
	return func(ctx context.Context, poolID string) (*chain.StakePool, error) {
		poolMetadata, err := b.client.PoolMetadata(ctx, poolID)
		if err != nil {
			return nil, err
		}
		return &chain.StakePool{
			HexID:  poolMetadata.Hex,
			Ticker: poolMetadata.Ticker,
			Name:   poolMetadata.Name,
		}, err
	}
}

func (b *Backend) GetMintedBlock(ctx context.Context,
	slot uint) (*chain.MintedBlock, error) {

	block, err := b.client.BlockBySlot(ctx, int(slot))
	if err != nil {
		if serr, ok := err.(*blockfrost.APIError); ok {
			if _, ok := serr.Response.(blockfrost.NotFound); ok {
				return nil, nil
			}
		}
		return nil, err
	}
	pool, err := b.cache.fetchPoolMetadata(ctx, block.SlotLeader,
		b.fetchPoolMetadata())
	if err != nil {
		return nil, err
	}
	return &chain.MintedBlock{
		Tip: chain.Tip{
			Height:      uint(block.Height),
			Hash:        block.Hash,
			Epoch:       uint(block.Epoch),
			SlotInEpoch: uint(block.EpochSlot),
			Slot:        uint(block.Slot),
			Timestamp:   uint(block.Time),
		},
		Pool: *pool,
	}, nil
}

func (b *Backend) TraverseAround(ctx context.Context, slot uint,
	interval uint) (*chain.MintedBlock, error) {

	for i := 1; i <= int(interval); i++ {
		mBlock, err := b.GetMintedBlock(ctx, slot+uint(i))
		if err != nil {
			return nil, err
		}
		if mBlock != nil {
			return mBlock, nil
		}
		mBlock, err = b.GetMintedBlock(ctx, slot-uint(i))
		if err != nil {
			return nil, err
		}
		if mBlock != nil {
			return mBlock, nil
		}
	}
	return nil, nil
}
