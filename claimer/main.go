package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	brpc "github.com/binance-chain/go-sdk/client/rpc"
	btypes "github.com/binance-chain/go-sdk/common/types"
	bkeys "github.com/binance-chain/go-sdk/keys"
	amino "github.com/tendermint/go-amino"

	sdk "github.com/kava-labs/cosmos-sdk/types"
	authtypes "github.com/kava-labs/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/tendermint/libs/log"
	rpcclient "github.com/kava-labs/tendermint/rpc/client"
	tmtypes "github.com/kava-labs/tendermint/types"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

type KavaClaimer struct {
	Keybase      keys.KeyManager
	LastBlockNum int64
	Status       bool
}

func main() {
	// Load config
	config, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	c := *config

	// Load kava claimers
	sdkConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(sdkConfig)
	cdc := kava.MakeCodec()

	var kavaClaimers []KavaClaimer
	for _, kavaMnemonic := range c.Kava.Mnemonics {
		kavaClaimer := KavaClaimer{}
		keyManager, err := keys.NewMnemonicKeyManager(kavaMnemonic, kava.Bip44CoinType)
		if err != nil {
			fmt.Println(err)
		}
		kavaClaimer.Keybase = keyManager
		kavaClaimer.LastBlockNum = 0
		kavaClaimer.Status = true
		kavaClaimers = append(kavaClaimers, kavaClaimer)
	}

	// Set up Kava HTTP client
	http, err := rpcclient.NewHTTP(c.Kava.Endpoint, "/websocket")
	if err != nil {
		panic(err)
	}
	http.Logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Set up Binance Chain client
	bnbClient := brpc.NewRPCClient(c.BinanceChain.Endpoint, btypes.ProdNetwork) // TODO:
	bnbKeyManager, err := bkeys.NewMnemonicKeyManager(c.BinanceChain.Mnemonic)
	if err != nil {
		panic(err)
	}
	bnbClient.SetKeyManager(bnbKeyManager)

	fmt.Println("Starting server...")
	claims := make(chan server.ClaimJob, 1000)
	server := server.NewServer(claims)
	go server.StartServer()

	for {
		select {
		case claim := <-claims:
			switch strings.ToUpper(claim.TargetChain) {
			case "KAVA":
				claimOnKava(http, claim, cdc, kavaClaimers)
			case "BINANCE", "BINANCE CHAIN":
				claimOnBinanceChain(bnbClient, claim)
			default:
				fmt.Println("invalid target chain:", claim.TargetChain)
			}
		}
	}
}

func claimOnBinanceChain(bnbHTTP brpc.Client, claim server.ClaimJob) {
	swapID, err := hex.DecodeString(claim.SwapID)
	if err != nil {
		fmt.Println(err)
	}

	canClaim := false
	swap, err := bnbHTTP.GetSwapByID(swapID[:])

	if err != nil {
		if strings.Contains(err.Error(), "zero records") {
			fmt.Println(fmt.Sprintf("swap %s not found", claim.SwapID))
		}
		fmt.Println(err)
	}

	status, err := bnbHTTP.Status()

	if err != nil {
		fmt.Println(err)
	}

	if swap.Status == btypes.Open && status.SyncInfo.LatestBlockHeight < swap.ExpireHeight {
		canClaim = true
	} else {
		fmt.Println(fmt.Sprintf("Swap has status %s and cannot be claimed", swap.Status))
	}

	if canClaim {
		randomNumber, err := hex.DecodeString(claim.RandomNumber)
		if err != nil {
			fmt.Println(err)
		}

		res, err := bnbHTTP.ClaimHTLT(swapID[:], randomNumber[:], brpc.Sync)
		if err != nil {
			fmt.Println(err)
		}

		if res.Code != 0 {
			fmt.Println(errors.New(res.Log))
		}

		fmt.Println("Tx hash:", res.Hash.String())
	}
}

func claimOnKava(http *rpcclient.HTTP, claim server.ClaimJob, cdc *amino.Codec, kavaClaimers []KavaClaimer) error {
	// Get current chain height
	abciInfo, err := http.ABCIInfo()
	if err != nil {
		return err
	}
	lastBlockHeight := abciInfo.Response.LastBlockHeight

	swapID, err := hex.DecodeString(claim.SwapID)
	if err != nil {
		return err
	}

	claimable, err := isClaimableKava(http, cdc, swapID)
	if err != nil {
		return err // TODO: allow for retrying
	}

	if !claimable {
		return errors.New("swap is not claimable")
	}

	var claimer KavaClaimer
	selectedClaimer := false
	for !selectedClaimer {
		source := rand.NewSource(time.Now().UnixNano())
		r := rand.New(source)
		// TODO: could use comparable here instead of random
		randNumb := r.Intn(len(kavaClaimers))
		randClaimer := kavaClaimers[randNumb%len(kavaClaimers)]
		if randClaimer.Status && randClaimer.LastBlockNum <= lastBlockHeight {
			selectedClaimer = true
			claimer = randClaimer
		}
	}

	fromAddr := claimer.Keybase.GetAddr()

	randomNumber, err := hex.DecodeString(claim.RandomNumber)
	if err != nil {
		return err
	}

	msg := bep3.NewMsgClaimAtomicSwap(fromAddr, swapID, randomNumber)
	if err := msg.ValidateBasic(); err != nil {
		return fmt.Errorf("msg basic validation failed: \n%v", msg)
	}

	signMsg := &authtypes.StdSignMsg{
		ChainID:       "testing", // TODO: customizable chain ID
		AccountNumber: 0,
		Sequence:      0,
		Fee:           authtypes.NewStdFee(200000, sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(250000)))),
		Msgs:          []sdk.Msg{msg},
		Memo:          "",
	}

	sequence, accountNumber, err := getKavaAcc(http, cdc, fromAddr)
	if err != nil {
		return err
	}
	signMsg.Sequence = sequence
	signMsg.AccountNumber = accountNumber

	signedMsg, err := claimer.Keybase.Sign(*signMsg, cdc)
	if err != nil {
		return err
	}
	tx := tmtypes.Tx(signedMsg)

	maxTxLength := 1024 * 1024
	if len(tx) > maxTxLength {
		return fmt.Errorf("the tx data exceeds max length %d ", maxTxLength)
	}

	txRes, err := http.BroadcastTxSync(tx)
	if err != nil {
		return err
	}

	// Update block height to prevent this claimer from being used again this block
	claimer.LastBlockNum = lastBlockHeight + 1

	fmt.Println("Tx hash:", txRes.Hash, " code:", txRes.Code)
	return nil
}

// Check if swap is claimable
func isClaimableKava(http *rpcclient.HTTP, cdc *amino.Codec, swapID []byte) (bool, error) {
	claimableParams := bep3.NewQueryAtomicSwapByID(swapID)
	claimableBz, err := cdc.MarshalJSON(claimableParams)
	if err != nil {
		return false, err
	}

	claimablePath := "custom/bep3/swap"

	result, err := http.ABCIQuery(claimablePath, claimableBz)
	if err != nil {
		return false, err
	}

	resp := result.Response
	if !resp.IsOK() {
		return false, errors.New(resp.Log)
	}

	value := result.Response.GetValue()
	if len(value) == 0 {
		return false, errors.New("no response value")
	}

	var swap bep3.AtomicSwap
	err = cdc.UnmarshalJSON(value, &swap)
	if err != nil {
		return false, err
	}

	if swap.Status == bep3.NULL {
		return false, fmt.Errorf("Swap %s not found", swapID)
	} else if swap.Status != bep3.Open {
		return false, fmt.Errorf("Swap has status %s and cannot be claimed", swap.Status)
	}

	return true, nil
}

func getKavaAcc(http *rpcclient.HTTP, cdc *amino.Codec, fromAddr sdk.AccAddress) (uint64, uint64, error) {
	params := authtypes.NewQueryAccountParams(fromAddr)
	bz, err := cdc.MarshalJSON(params)
	if err != nil {
		return 0, 0, err
	}

	path := fmt.Sprintf("custom/acc/account/%s", fromAddr.String())

	result, err := http.ABCIQuery(path, bz)
	if err != nil {
		return 0, 0, err
	}

	resp := result.Response
	if !resp.IsOK() {
		return 0, 0, errors.New(resp.Log)
	}

	value := result.Response.GetValue()
	if len(value) == 0 {
		return 0, 0, errors.New("no response value")
	}

	var acc authtypes.BaseAccount
	err = cdc.UnmarshalJSON(value, &acc)
	if err != nil {
		return 0, 0, err
	}

	if acc.Address.Empty() {
		return 0, 0, errors.New("the signer account does not exist on kava")
	}

	return acc.Sequence, acc.AccountNumber, nil
}
