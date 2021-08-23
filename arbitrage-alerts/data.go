package main

import (
	swaptypes "github.com/kava-labs/kava/x/swap/types"
)

type SwapData struct {
	PoolStats swaptypes.PoolStatsQueryResults
}

func GetSwapPoolData(client SwapClient) (*SwapData, error) {
	info, err := client.GetInfo()
	if err != nil {
		return &SwapData{}, err
	}

	height := info.LatestHeight

	poolStats, err := client.GetPools(height)
	if err != nil {
		return &SwapData{}, err
	}

	return &SwapData{
		PoolStats: poolStats,
	}, nil
}
