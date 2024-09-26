package auctions

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuctionsTotal(t *testing.T) {
	usdValue, err := sdk.NewDecFromStr("0.000001705763333334")
	require.NoError(t, err)

	data, err := getAuctionDataAtHeight(dataQueryClient, 11800020)
	require.NoError(t, err)

	totalValue, err := CalculateTotalAuctionsUSDValue(data)
	require.NoError(t, err)

	require.True(t, totalValue.Equal(usdValue))
}
