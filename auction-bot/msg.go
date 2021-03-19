package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
)

func CreateBidMsgs(keeper sdk.AccAddress, auctionBids AuctionInfos) []auctiontypes.MsgPlaceBid {
	msgs := make([]auctiontypes.MsgPlaceBid, len(auctionBids))

	for index, bid := range auctionBids {
		msgs[index] = auctiontypes.NewMsgPlaceBid(bid.ID, bid.Bidder, bid.Amount)
	}

	return msgs
}
