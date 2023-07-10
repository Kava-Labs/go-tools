package types

// ClaimJob defines a claim request received by the server
type ClaimJob struct {
	ID           string
	TargetChain  string
	SwapID       string
	RandomNumber string
}

// NewClaimJob instantiates a new instance of ClaimJob
func NewClaimJob(id, targetChain, swapID, randomNumber string) ClaimJob {
	return ClaimJob{
		ID:           id,
		TargetChain:  targetChain,
		SwapID:       swapID,
		RandomNumber: randomNumber,
	}
}
