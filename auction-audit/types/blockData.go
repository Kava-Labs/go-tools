package types

import (
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type BlockData struct {
	Block       *tmproto.Block
	BlockResult *ctypes.ResultBlockResults
}
