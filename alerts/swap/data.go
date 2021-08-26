package swap

import (
	"fmt"

	swaptypes "github.com/kava-labs/kava/x/swap/types"
)

// GetPoolsData returns current swap pools
func GetPoolsData(client SwapClient) (swaptypes.PoolStatsQueryResults, error) {
	// fetch chain info to get height
	info, err := client.GetInfo()
	if err != nil {
		return nil, err
	}

	// use height to get consistent state from rpc client
	height := info.LatestHeight

	prices, err := client.GetPrices(height)
	if err != nil {
		return nil, err
	}

	fmt.Println(prices)

	pools, err := client.GetPools(height)
	if err != nil {
		return nil, err
	}

	return pools, nil
}
