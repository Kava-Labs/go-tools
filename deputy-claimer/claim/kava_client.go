package claim

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/kava-labs/cosmos-sdk/client/rpc"
	"github.com/kava-labs/cosmos-sdk/codec"
	sdk "github.com/kava-labs/cosmos-sdk/types"
	authexported "github.com/kava-labs/cosmos-sdk/x/auth/exported"
	authTypes "github.com/kava-labs/cosmos-sdk/x/auth/types"
	kavaRpc "github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	tmbytes "github.com/kava-labs/tendermint/libs/bytes"
	tmRPCTypes "github.com/kava-labs/tendermint/rpc/core/types"
	tmtypes "github.com/kava-labs/tendermint/types"
)

type kavaChainClient struct {
	restURL, rpcURL string
	codec           *codec.Codec
	kavaSDKClient   *kavaRpc.KavaClient
}

func newKavaChainClient(restURL, rpcURL string, cdc *codec.Codec) kavaChainClient {
	// use a fake mnemonic as we're not using the kava client for signing
	dummyMnemonic := "adult stem bus people vast riot eager faith sponsor unlock hold lion sport drop eyebrow loud angry couch panic east three credit grain talk"
	return kavaChainClient{
		restURL:       restURL,
		codec:         cdc,
		kavaSDKClient: kavaRpc.NewKavaClient(cdc, dummyMnemonic, kava.Bip44CoinType, rpcURL, kavaRpc.ProdNetwork),
	}
}

type restResponse struct {
	Height int             `json:"height"`
	Result json.RawMessage `json:"result"`
}

func (kc kavaChainClient) getOpenSwaps() (bep3.AtomicSwaps, error) {
	resp, err := http.Get(kc.restURL + "/bep3/swaps?direction=outgoing&status=open&limit=1000")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res restResponse
	kc.codec.MustUnmarshalJSON(bz, &res)
	var swaps bep3.AtomicSwaps
	kc.codec.MustUnmarshalJSON(res.Result, &swaps)
	return swaps, nil
}

func (kc kavaChainClient) getAccount(address sdk.AccAddress) (authexported.Account, error) {
	resp, err := http.Get(kc.restURL + "/auth/accounts/" + address.String()) // TODO construct urls properly
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res restResponse
	kc.codec.MustUnmarshalJSON(bz, &res)
	var account authexported.Account
	kc.codec.MustUnmarshalJSON(res.Result, &account)
	return account, nil
}

func (kc kavaChainClient) getTxConfirmation(txHash []byte) (*tmRPCTypes.ResultTx, error) {
	return kc.kavaSDKClient.HTTP.Tx(txHash, false)
}

func (kc kavaChainClient) broadcastTx(tx tmtypes.Tx) error {
	res, err := kc.kavaSDKClient.BroadcastTxSync(tx)
	if err != nil {
		return err
	}
	if res.Code != 0 { // tx failed to be submitted to the mempool
		return fmt.Errorf("transaction failed to get into mempool: %s", res.Log)
	}
	return nil
}

func (kc kavaChainClient) getChainID() (string, error) {
	infoResp, err := http.Get(kc.restURL + "/node_info")
	if err != nil {
		return "", err
	}
	defer infoResp.Body.Close()
	infoBz, err := ioutil.ReadAll(infoResp.Body)
	if err != nil {
		return "", err
	}
	var nodeInfo rpc.NodeInfoResponse
	kc.codec.MustUnmarshalJSON(infoBz, &nodeInfo)
	return nodeInfo.Network, nil
}

func (kc kavaChainClient) getSwapByID(id tmbytes.HexBytes) (bep3.AtomicSwap, error) {
	return kc.kavaSDKClient.GetSwapByID(id)
}

func (kc kavaChainClient) getRandomNumberFromSwap(id []byte) (tmbytes.HexBytes, error) {
	strID := strings.ToLower(hex.EncodeToString(id))
	query := fmt.Sprintf(`claim_atomic_swap.atomic_swap_id='%s'`, strID) // must be lowercase hex for querying to work
	res, err := kc.kavaSDKClient.HTTP.TxSearch(query, false, 1, 1000, "")
	if err != nil {
		return nil, err
	}
	if len(res.Txs) < 1 {
		return nil, fmt.Errorf("no claim txs found")
	}
	var stdTx authTypes.StdTx
	err = kc.codec.UnmarshalBinaryLengthPrefixed(res.Txs[0].Tx, &stdTx) // TODO handle case of there being more than one tx
	if err != nil {
		return nil, err
	}
	claim, ok := stdTx.Msgs[0].(bep3.MsgClaimAtomicSwap) // TODO handle the case of multiple messages
	if !ok {
		return nil, fmt.Errorf("unable to decode msg into MsgClaimAtomicSwap")
	}
	return claim.RandomNumber, nil
}
