package integrationtest

import (
	"fmt"
	"time"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbKeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/binance-chain-go-sdk/types/msg"
	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
)

// KavaSwap is a struct to hold parameters for creating a HTLT on the kava chain.
type KavaSwap struct {
	bep3types.AtomicSwap // TODO inline fields
	SenderMnemonic       string
	HeightSpan           uint64
}

// TODO add New(Outgoing/Incoming)KavaSwap methods?

func NewKavaSwap(senderMnemonic string, recipient sdk.AccAddress, senderOtherChain, recipientOtherChain string, amount sdk.Coins, timestamp int64, rndHash []byte, heightspan int64) KavaSwap {
	if heightspan < 0 {
		panic("heightspan cannot be negative")
	}
	return KavaSwap{
		AtomicSwap: bep3types.AtomicSwap{
			Amount:              amount,
			RandomNumberHash:    rndHash,
			Timestamp:           timestamp,
			Sender:              kavaAddressFromMnemonic(senderMnemonic),
			Recipient:           recipient,
			SenderOtherChain:    senderOtherChain,
			RecipientOtherChain: recipientOtherChain,
		},
		SenderMnemonic: senderMnemonic,
		HeightSpan:     uint64(heightspan),
	}
}

// BnbSwap is a struct to hold parameters for creating a HTLT on the bnb chain.
type BnbSwap struct {
	types.AtomicSwap
	SenderMnemonic   string
	SenderOtherChain string
	HeightSpan       int64
}

func NewBnbSwap(senderMnemonic string, recipient types.AccAddress, senderOtherChain, recipientOtherChain string, amount types.Coins, timestamp int64, rndHash []byte, heightSpan int64) BnbSwap {
	return BnbSwap{
		AtomicSwap: types.AtomicSwap{
			From:                bnbAddressFromMnemonic(senderMnemonic),
			To:                  recipient,
			RecipientOtherChain: recipientOtherChain,
			InAmount:            amount,
			RandomNumberHash:    rndHash,
			Timestamp:           timestamp,
			CrossChain:          true,
		},
		SenderMnemonic:   senderMnemonic,
		SenderOtherChain: senderOtherChain,
		HeightSpan:       heightSpan,
	}
}

func (swap BnbSwap) GetSwapID() []byte {
	return msg.CalculateSwapID(swap.RandomNumberHash, swap.From, swap.SenderOtherChain)
}

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

// CrossChainSwap holds details of both swaps involved in moving assets from one chain to the other.
type CrossChainSwap struct {
	KavaSwap     KavaSwap
	BnbSwap      BnbSwap
	RandomNumber []byte
}

// NewBnbToKavaSwap creates valid bnb and kava swaps to move assets from bnb to kava chains.
func NewBnbToKavaSwap(senderMnemonic string, recipient sdk.AccAddress, amount SwapAmount, kavaDeputyMnemonic string, bnbDeputyAddress types.AccAddress, rndHash []byte, timestamp int64, heightSpan SwapHeightSpan, rndNum []byte) CrossChainSwap {
	return CrossChainSwap{
		BnbSwap: NewBnbSwap(
			senderMnemonic,
			bnbDeputyAddress,
			kavaAddressFromMnemonic(kavaDeputyMnemonic).String(),
			recipient.String(),
			amount.Bnb,
			timestamp,
			rndHash,
			heightSpan.Bnb,
		),
		KavaSwap: NewKavaSwap(
			kavaDeputyMnemonic,
			recipient,
			bnbAddressFromMnemonic(senderMnemonic).String(),
			bnbDeputyAddress.String(),
			amount.Kava,
			timestamp,
			rndHash,
			heightSpan.Kava,
		),
		RandomNumber: rndNum,
	}
}

// NewKavaToBnbSwap creates valid kava and bnb swaps to move assets from kava to bnb chains.
func NewKavaToBnbSwap(senderMnemonic string, recipient types.AccAddress, amount SwapAmount, bnbDeputyMnemonic string, kavaDeputyAddress sdk.AccAddress, rndHash []byte, timestamp int64, heightSpan SwapHeightSpan, rndNum []byte) CrossChainSwap {
	return CrossChainSwap{
		KavaSwap: NewKavaSwap(
			senderMnemonic,
			kavaDeputyAddress,
			bnbAddressFromMnemonic(bnbDeputyMnemonic).String(),
			recipient.String(),
			amount.Kava,
			timestamp,
			rndHash,
			heightSpan.Kava,
		),
		BnbSwap: NewBnbSwap(
			bnbDeputyMnemonic,
			recipient,
			kavaAddressFromMnemonic(senderMnemonic).String(),
			kavaDeputyAddress.String(),
			amount.Bnb,
			timestamp,
			rndHash,
			heightSpan.Bnb,
		),
		RandomNumber: rndNum,
	}
}

// TODO create useful constructors for this to convert a single swap amount into correct denoms and take off deputy fee
type SwapAmount struct {
	Kava sdk.Coins
	Bnb  types.Coins
}

type SwapHeightSpan struct {
	Kava, Bnb int64
}

// SwapBuilder assists in creating cross chain swaps by storing common swap parameters.
type SwapBuilder struct {
	kavaDeputyMnemonic  string
	bnbDeputyMnemonic   string
	calculateKavaAmount func(types.Coins) sdk.Coins
	calculateBnbAmount  func(sdk.Coins) types.Coins
	heightSpanKavaToBnb SwapHeightSpan
	heightSpanBnbToKava SwapHeightSpan
	timestamper         func() int64
	// TODO add rand seed, or rand num generator func
}

// NewDefaultSwapBuilder creates a SwapBuilder with defaults for common swap parameters.
func NewDefaultSwapBuilder(kavaDeputyMnemonic, bnbDeputyMnemonic string) SwapBuilder {
	return SwapBuilder{
		kavaDeputyMnemonic:  kavaDeputyMnemonic,
		bnbDeputyMnemonic:   bnbDeputyMnemonic,
		calculateKavaAmount: convertBnbToKavaCoins,
		calculateBnbAmount:  convertKavaToBnbCoins,
		heightSpanKavaToBnb: SwapHeightSpan{
			Bnb:  360,
			Kava: 250,
		},
		heightSpanBnbToKava: SwapHeightSpan{ // TODO
			Bnb:  360,
			Kava: 250,
		},
		timestamper: getCurrentTimestamp,
	}
}

// WithTimestamp returns a SwapBuilder with a fixed value for swap timestamps.
func (builder SwapBuilder) WithTimestamp(timestamp int64) SwapBuilder {
	builder.timestamper = func() int64 { return timestamp }
	return builder
}

func (builder SwapBuilder) NewBnbToKavaSwap(senderMnemonic string, recipient sdk.AccAddress, amount types.Coins) CrossChainSwap {
	rndNum, err := bep3types.GenerateSecureRandomNumber()
	if err != nil {
		panic(err)
	}
	timestamp := builder.timestamper()
	rndHash := bep3types.CalculateRandomHash(rndNum, timestamp)
	return NewBnbToKavaSwap(
		senderMnemonic,
		recipient,
		SwapAmount{
			Bnb:  amount,
			Kava: builder.calculateKavaAmount(amount),
		},
		builder.kavaDeputyMnemonic,
		bnbAddressFromMnemonic(builder.bnbDeputyMnemonic),
		rndHash,
		timestamp,
		builder.heightSpanBnbToKava,
		rndNum,
	)
}
func (builder SwapBuilder) NewKavaToBnbSwap(senderMnemonic string, recipient types.AccAddress, amount sdk.Coins) CrossChainSwap {
	rndNum, err := bep3types.GenerateSecureRandomNumber()
	if err != nil {
		panic(err)
	}
	timestamp := builder.timestamper()
	rndHash := bep3types.CalculateRandomHash(rndNum, timestamp)
	return NewKavaToBnbSwap(
		senderMnemonic,
		recipient,
		SwapAmount{
			Kava: amount,
			Bnb:  builder.calculateBnbAmount(amount),
		},
		builder.bnbDeputyMnemonic,
		kavaAddressFromMnemonic(builder.kavaDeputyMnemonic),
		rndHash,
		timestamp,
		builder.heightSpanKavaToBnb,
		rndNum,
	)
}

func getCurrentTimestamp() int64 { return time.Now().Unix() }

var denomMap = map[string]string{
	"XRP-BF2":  "xrpb",
	"BUSD-BD1": "busd",
	"BTCB-1DE": "btcb",
	"BNB":      "bnb",
}

func convertBnbToKavaCoins(coins types.Coins) sdk.Coins {
	sdkCoins := sdk.NewCoins()
	for _, c := range coins {
		newDenom, ok := denomMap[c.Denom]
		if !ok {
			panic(fmt.Sprintf("unrecognized coin denom '%s'", c.Denom))
		}
		sdkCoins = sdkCoins.Add(sdk.NewInt64Coin(newDenom, c.Amount))
	}
	return sdkCoins
}
func convertKavaToBnbCoins(coins sdk.Coins) types.Coins {
	bnbCoins := types.Coins{}
	for _, c := range coins {
		newDenom, ok := reverseStringMap(denomMap)[c.Denom]
		if !ok {
			panic(fmt.Sprintf("unrecognized coin denom '%s'", c.Denom))
		}
		if !c.Amount.IsInt64() {
			panic(fmt.Sprintf("coin amount '%s' cannot be converted to int64", c.Amount))
		}
		bnbCoins = bnbCoins.Plus(types.Coins{types.Coin{Denom: newDenom, Amount: c.Amount.Int64()}})
	}
	return bnbCoins.Sort()
}

func kavaAddressFromMnemonic(mnemonic string) sdk.AccAddress {
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
	if err != nil {
		panic(fmt.Sprintf("new key manager from mnenomic err, err=%s", err.Error())) // TODO
	}
	return keyManager.GetAddr()
}
func bnbAddressFromMnemonic(mnemonic string) types.AccAddress {
	keyManager, err := bnbKeys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		panic(err)
	}
	return keyManager.GetAddr()
}

func reverseStringMap(m map[string]string) map[string]string {
	reversedMap := make(map[string]string, len(m))
	for k, v := range m {
		reversedMap[v] = k
	}
	return reversedMap
}
