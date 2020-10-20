package integrationtest

import (
	"fmt"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbKeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
)

// KavaSwapClient handles sending txs to modify a kava swap on chain.
// It can create, claim, or refund a swap.
type KavaSwapClient struct {
	kavaRpcUrl string
}

func NewKavaSwapClient(KavaRpcUrl string) KavaSwapClient {
	return KavaSwapClient{kavaRpcUrl: KavaRpcUrl}
}
func (swapper KavaSwapClient) Create(swap KavaSwap, mode client.SyncType) ([]byte, error) {
	msg := bep3types.NewMsgCreateAtomicSwap(
		swap.Sender,
		swap.Recipient,
		swap.RecipientOtherChain,
		swap.SenderOtherChain,
		swap.RandomNumberHash,
		swap.Timestamp,
		swap.Amount,
		swap.HeightSpan,
	)
	return swapper.broadcastMsg(msg, swap.SenderMnemonic, mode)
}
func (swapper KavaSwapClient) Claim(swap KavaSwap, randomNumber []byte, mode client.SyncType) ([]byte, error) {
	msg := bep3types.NewMsgClaimAtomicSwap(
		swap.Sender, // doesn't need to be sender
		swap.GetSwapID(),
		randomNumber,
	)
	return swapper.broadcastMsg(msg, swap.SenderMnemonic, mode)
}
func (swapper KavaSwapClient) Refund(swap KavaSwap, mode client.SyncType) ([]byte, error) {
	msg := bep3types.NewMsgRefundAtomicSwap(
		swap.Sender, // doesn't need to be sender
		swap.GetSwapID(),
	)
	return swapper.broadcastMsg(msg, swap.SenderMnemonic, mode)
}
func (swapper KavaSwapClient) FetchStatus(swap KavaSwap) (bep3types.SwapStatus, error) {
	standInMnemonic := "grass luxury welcome dismiss legal nothing glide crisp material broccoli jewel put inflict expose taxi wear second party air hockey crew ride wage nurse"
	kavaClient := client.NewKavaClient(app.MakeCodec(), standInMnemonic, app.Bip44CoinType, swapper.kavaRpcUrl)
	fetchedSwap, err := kavaClient.GetSwapByID(swap.GetSwapID())
	if err != nil {
		return 0, fmt.Errorf("could not fetch swap status: %w", err)
	}
	return fetchedSwap.Status, nil
}

func (swapper KavaSwapClient) broadcastMsg(msg sdk.Msg, signerMnemonic string, mode client.SyncType) ([]byte, error) {
	cdc := app.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, signerMnemonic, app.Bip44CoinType, swapper.kavaRpcUrl)

	res, err := kavaClient.Broadcast(msg, mode)
	if err != nil {
		return res.Hash, fmt.Errorf("swap rejected from node: %w", err)
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}

// BnbSwapClient handles sending txs to modify a bnb swap on chain.
// It can create, claim, or refund a swap.
type BnbSwapClient struct {
	bnbSdkClient *bnbRpc.HTTP
}

func NewBnbSwapClient(bnbRpcUrl string) BnbSwapClient {
	return BnbSwapClient{
		bnbSdkClient: bnbRpc.NewRPCClient(bnbRpcUrl, types.ProdNetwork),
	}
}

func (swapper BnbSwapClient) Create(swap BnbSwap, mode bnbRpc.SyncType) ([]byte, error) {
	swapper.setSigningKey(swap.SenderMnemonic)
	res, err := swapper.bnbSdkClient.HTLT(
		swap.To,
		swap.RecipientOtherChain,
		swap.SenderOtherChain,
		swap.RandomNumberHash,
		swap.Timestamp,
		swap.InAmount,
		swap.ExpectedIncome,
		swap.HeightSpan,
		swap.CrossChain,
		mode,
	)
	if err != nil {
		return res.Hash, fmt.Errorf("swap rejected from node: %w", err)
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}

func (swapper BnbSwapClient) Claim(swap BnbSwap, randomNumber []byte, mode bnbRpc.SyncType) ([]byte, error) {
	swapper.setSigningKey(swap.SenderMnemonic)
	res, err := swapper.bnbSdkClient.ClaimHTLT(swap.GetSwapID(), randomNumber, mode)
	if err != nil {
		return res.Hash, fmt.Errorf("swap rejected from node: %w", err)
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}

func (swapper BnbSwapClient) Refund(swap BnbSwap, mode bnbRpc.SyncType) ([]byte, error) {
	swapper.setSigningKey(swap.SenderMnemonic)
	res, err := swapper.bnbSdkClient.RefundHTLT(swap.GetSwapID(), mode)
	if err != nil {
		return res.Hash, fmt.Errorf("swap rejected from node: %w", err)
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}

func (swapper BnbSwapClient) FetchStatus(swap BnbSwap) (types.SwapStatus, error) {
	fetchedSwap, err := swapper.bnbSdkClient.GetSwapByID(swap.GetSwapID())
	if err != nil {
		return 0, fmt.Errorf("could not fetch swap status: %w", err)
	}
	return fetchedSwap.Status, nil
}

func (swapper BnbSwapClient) setSigningKey(mnemonic string) {
	bnbKeyM, err := bnbKeys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		panic(err)
	}
	swapper.bnbSdkClient.SetKeyManager(bnbKeyM)
}
