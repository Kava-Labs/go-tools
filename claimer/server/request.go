package server

const (
	RestTargetChain  = "target-chain"
	RestSwapID       = "swap-id"
	RestRandomNumber = "random-number"
)

type PostClaimSwapReq struct {
	TargetChain  string `json:"target_chain" yaml:"target_chain"`
	SwapID       []byte `json:"swap_id" yaml:"swap_id"`
	RandomNumber []byte `json:"random_number" yaml:"random_number"`
}
