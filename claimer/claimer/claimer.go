package claimer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	brpc "github.com/binance-chain/go-sdk/client/rpc"
	btypes "github.com/binance-chain/go-sdk/common/types"
	ec "github.com/ethereum/go-ethereum/common"

	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/go-sdk/kava/bep3"
)

// Claimer sends claim transactions on Kava and Binance Chain
type Claimer struct {
	kavaClient *client.KavaClient
	bncClient  brpc.Client
}

// NewClaimer instantiates a new instance of Claimer
func NewClaimer(kavaClient *client.KavaClient, bncClient brpc.Client) Claimer {
	return Claimer{
		kavaClient: kavaClient,
		bncClient:  bncClient,
	}
}

func (c Claimer) IsClaimableKava(swapID []byte) (bool, error) {
	swap, err := c.kavaClient.GetSwapByID(swapID)
	if err != nil {
		return false, err
	}

	if swap.Status == bep3.Open {
		return true, nil
	}

	return false, nil
}

func (c Claimer) SendClaimKava(rawSwapID, rawRandomNumber string) (string, error) {
	swapID, err := hex.DecodeString(rawSwapID)
	if err != nil {
		return "", err
	}

	msg := bep3.NewMsgClaimAtomicSwap(c.kavaClient.Keybase.GetAddr(), swapID, []byte(rawRandomNumber))
	if err := msg.ValidateBasic(); err != nil {
		return "", fmt.Errorf("msg basic validation failed: \n%v", msg)
	}

	res, err := c.kavaClient.Broadcast(msg, client.Sync)
	if err != nil {
		return "", err
	}
	if res.Code != 0 {
		return "", fmt.Errorf("\nres.Code: %d\nLog:%s", res.Code, res.Log)

	}

	return res.Hash.String(), nil
}

func (c Claimer) IsClaimableBinance(swapID ec.Hash) (bool, error) {
	swap, err := c.bncClient.GetSwapByID(swapID[:])
	if err != nil {
		if strings.Contains(err.Error(), "zero records") {
			return false, nil
		}
		return false, err
	}

	status, err := c.bncClient.Status()
	if err != nil {
		return false, err
	}

	if swap.Status == btypes.Open && status.SyncInfo.LatestBlockHeight < swap.ExpireHeight {
		return true, nil
	}

	return false, nil
}

func (c Claimer) SendClaimBinance(rawSwapID, randomNumber string) (string, error) {
	swapID, err := hex.DecodeString(rawSwapID)
	if err != nil {
		return "", err
	}

	res, err := c.bncClient.ClaimHTLT(swapID[:], []byte(randomNumber[:]), brpc.Sync)
	if err != nil {
		return "", err
	}
	if res.Code != 0 {
		return "", errors.New(res.Log)
	}

	return res.Hash.String(), nil
}
