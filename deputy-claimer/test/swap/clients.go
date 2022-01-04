package swap

import (
	"context"
	"fmt"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
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

func (swapClient KavaSwapClient) Create(swap KavaSwap, mode client.SyncType) (string, error) {
	msg := bep3types.NewMsgCreateAtomicSwap(
		swap.Sender.String(),
		swap.Recipient.String(),
		swap.RecipientOtherChain,
		swap.SenderOtherChain,
		swap.RandomNumberHash,
		swap.Timestamp,
		swap.Amount,
		swap.HeightSpan,
	)

	return swapClient.broadcastMsg(&msg, swap.SenderMnemonic, mode)
}

func (swapClient KavaSwapClient) Claim(swap KavaSwap, randomNumber []byte, mode client.SyncType) (string, error) {
	msg := bep3types.NewMsgClaimAtomicSwap(
		swap.Sender.String(), // doesn't need to be sender
		swap.GetSwapID(),
		randomNumber,
	)

	return swapClient.broadcastMsg(&msg, swap.SenderMnemonic, mode)
}

func (swapClient KavaSwapClient) Refund(swap KavaSwap, mode client.SyncType) (string, error) {
	msg := bep3types.NewMsgRefundAtomicSwap(
		swap.Sender.String(), // doesn't need to be sender
		swap.GetSwapID(),
	)

	return swapClient.broadcastMsg(&msg, swap.SenderMnemonic, mode)
}

func (swapClient KavaSwapClient) FetchStatus(swap KavaSwap) (bep3types.SwapStatus, error) {
	standInMnemonic := "grass luxury welcome dismiss legal nothing glide crisp material broccoli jewel put inflict expose taxi wear second party air hockey crew ride wage nurse"
	encodingConfig := app.MakeEncodingConfig()

	kavaClient := client.NewKavaClient(encodingConfig.Amino, standInMnemonic, app.Bip44CoinType, swapClient.kavaRpcUrl)
	fetchedSwap, err := kavaClient.GetSwapByID(context.Background(), swap.GetSwapID())
	if err != nil {
		return 0, fmt.Errorf("could not fetch swap status: %w", err)
	}
	return fetchedSwap.Status, nil
}

func (swapClient KavaSwapClient) broadcastMsg(msg sdk.Msg, signerMnemonic string, mode client.SyncType) (string, error) {
	encodingConfig := app.MakeEncodingConfig()
	kavaClient := client.NewKavaClient(encodingConfig.Amino, signerMnemonic, app.Bip44CoinType, swapClient.kavaRpcUrl)
	ctx := sdkclient.Context{}.
		WithClient(kavaClient.HTTP).
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino)

	res, err := kavaClient.Broadcast(ctx, msg, mode)
	if err != nil {
		return "", err
	}
	if res.Code != 0 {
		return res.TxHash, fmt.Errorf("tx rejected: %s", res.Logs)
	}
	return res.TxHash, nil
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

func (swapClient BnbSwapClient) Create(swap BnbSwap, mode bnbRpc.SyncType) ([]byte, error) {
	swapClient.setSigningKey(swap.SenderMnemonic)
	res, err := swapClient.bnbSdkClient.HTLT(
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
		return nil, fmt.Errorf("swap rejected from node: %w", err)
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}

func (swapClient BnbSwapClient) Claim(swap BnbSwap, randomNumber []byte, mode bnbRpc.SyncType) ([]byte, error) {
	swapClient.setSigningKey(swap.SenderMnemonic)
	res, err := swapClient.bnbSdkClient.ClaimHTLT(swap.GetSwapID(), randomNumber, mode)
	if err != nil {
		return res.Hash, fmt.Errorf("swap rejected from node: %w", err)
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}

func (swapClient BnbSwapClient) Refund(swap BnbSwap, mode bnbRpc.SyncType) ([]byte, error) {
	swapClient.setSigningKey(swap.SenderMnemonic)
	res, err := swapClient.bnbSdkClient.RefundHTLT(swap.GetSwapID(), mode)
	if err != nil {
		return res.Hash, fmt.Errorf("swap rejected from node: %w", err)
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}

func (swapClient BnbSwapClient) FetchStatus(swap BnbSwap) (types.SwapStatus, error) {
	fetchedSwap, err := swapClient.bnbSdkClient.GetSwapByID(swap.GetSwapID())
	if err != nil {
		return 0, fmt.Errorf("could not fetch swap status: %w", err)
	}
	return fetchedSwap.Status, nil
}

func (swapClient BnbSwapClient) setSigningKey(mnemonic string) {
	bnbKeyM, err := bnbKeys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		panic(err)
	}
	swapClient.bnbSdkClient.SetKeyManager(bnbKeyM)
}
