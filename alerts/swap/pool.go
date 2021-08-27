package swap

import "fmt"

func GetPools(pools SwapPoolsData) error {
	for _, pool := range pools.Pools {
		if len(pool.Coins) != 2 {
			return fmt.Errorf("Pool %v does not contain 2 coins", pool.Name)
		}

		first := pool.Coins[0]
		second := pool.Coins[1]

		firstUsdPrice, err := pools.BinancePrices.UsdValue(first.Denom, pools.UsdValues.Busd)
		if err != nil {
			return err
		}
	}
}
