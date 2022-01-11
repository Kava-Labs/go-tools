package swap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/go-tools/signing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	bnbKeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/app/params"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
)

// KavaSwapClient handles sending txs to modify a kava swap on chain.
// It can create, claim, or refund a swap.
type KavaSwapClient struct {
	encodingConfig params.EncodingConfig
	GrpcClientConn *grpc.ClientConn
	Auth           authtypes.QueryClient
	Tx             txtypes.ServiceClient
	Bep3           bep3types.QueryClient
}

func NewKavaSwapClient(target string) KavaSwapClient {
	grpcUrl, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
	}

	var secureOpt grpc.DialOption
	switch grpcUrl.Scheme {
	case "http":
		secureOpt = grpc.WithInsecure()
	case "https":
		creds := credentials.NewTLS(&tls.Config{})
		secureOpt = grpc.WithTransportCredentials(creds)
	default:
		log.Fatalf("unknown grpc url scheme %s\n", grpcUrl.Scheme)
	}

	grpcConn, err := grpc.Dial(grpcUrl.Host, secureOpt)
	if err != nil {
		panic(err)
	}

	return KavaSwapClient{
		encodingConfig: app.MakeEncodingConfig(),
		GrpcClientConn: grpcConn,
		Auth:           authtypes.NewQueryClient(grpcConn),
		Tx:             txtypes.NewServiceClient(grpcConn),
		Bep3:           bep3types.NewQueryClient(grpcConn),
	}
}

func (swapClient KavaSwapClient) Create(swap KavaSwap, mode txtypes.BroadcastMode) (string, error) {
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

func (swapClient KavaSwapClient) Claim(swap KavaSwap, randomNumber []byte, mode txtypes.BroadcastMode) (string, error) {
	msg := bep3types.NewMsgClaimAtomicSwap(
		swap.Sender.String(), // doesn't need to be sender
		swap.GetSwapID(),
		randomNumber,
	)

	return swapClient.broadcastMsg(&msg, swap.SenderMnemonic, mode)
}

func (swapClient KavaSwapClient) Refund(swap KavaSwap, mode txtypes.BroadcastMode) (string, error) {
	msg := bep3types.NewMsgRefundAtomicSwap(
		swap.Sender.String(), // doesn't need to be sender
		swap.GetSwapID(),
	)

	return swapClient.broadcastMsg(&msg, swap.SenderMnemonic, mode)
}

func (swapClient KavaSwapClient) FetchStatus(swap KavaSwap) (bep3types.SwapStatus, error) {
	res, err := swapClient.Bep3.AtomicSwap(context.Background(), &bep3types.QueryAtomicSwapRequest{
		SwapId: swap.GetSwapID().String(),
	})
	if err != nil {
		return 0, fmt.Errorf("could not fetch swap status: %w", err)
	}

	return res.AtomicSwap.Status, nil
}

func (swapClient KavaSwapClient) broadcastMsg(msg sdk.Msg, signerMnemonic string, mode txtypes.BroadcastMode) (string, error) {
	txBuilder := swapClient.encodingConfig.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		return "", err
	}
	txBuilder.SetGasLimit(250000)

	hdPath := hd.CreateHDPath(app.Bip44CoinType, 0, 0)
	privKeyBytes, err := hd.Secp256k1.Derive()(signerMnemonic, "", hdPath.String())
	if err != nil {
		panic(fmt.Sprintf("failed to derive key: %v", err))
	}

	// wrap with cosmos secp256k1 private key struct
	privKey := &secp256k1.PrivKey{Key: privKeyBytes}

	// Fetch account to get account number and sequence
	accRes, err := swapClient.Auth.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: signing.GetAccAddress(privKey).String(),
	})
	if err != nil {
		return "", err
	}

	var account authtypes.AccountI
	err = swapClient.encodingConfig.Marshaler.UnpackAny(accRes.Account, &account)
	if err != nil {
		return "", err
	}

	signerData := authsigning.SignerData{
		ChainID:       "kava-localnet",
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
	}

	_, txBytes, err := signing.Sign(swapClient.encodingConfig, privKey, txBuilder, signerData)
	if err != nil {
		return "", err
	}

	res, err := swapClient.Tx.BroadcastTx(context.Background(), &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    mode,
	})
	if err != nil {
		return "", err
	}
	if res.TxResponse.Code != 0 {
		return res.TxResponse.TxHash, fmt.Errorf("tx rejected with code %v: %v", res.TxResponse.Code, res)
	}

	return res.TxResponse.TxHash, nil
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
