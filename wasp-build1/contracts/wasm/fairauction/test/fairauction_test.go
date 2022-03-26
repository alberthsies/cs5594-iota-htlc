// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"testing"
	"time"

	"github.com/iotaledger/wasp/contracts/wasm/fairauction/go/fairauction"
	"github.com/iotaledger/wasp/packages/solo"
	"github.com/iotaledger/wasp/packages/wasmvm/wasmlib/go/wasmlib/wasmtypes"
	"github.com/iotaledger/wasp/packages/wasmvm/wasmsolo"
	"github.com/stretchr/testify/require"
)

var (
	auctioneer *wasmsolo.SoloAgent
	tokenColor wasmtypes.ScColor
)

func startAuction(t *testing.T) *wasmsolo.SoloContext {
	ctx := wasmsolo.NewSoloContext(t, fairauction.ScName, fairauction.OnLoad)

	// set up auctioneer account and mint some tokens to auction off
	auctioneer = ctx.NewSoloAgent()
	tokenColor, ctx.Err = auctioneer.Mint(10)
	require.NoError(t, ctx.Err)
	require.EqualValues(t, solo.Saldo-10, auctioneer.Balance())
	require.EqualValues(t, 10, auctioneer.Balance(tokenColor))

	// start the auction
	sa := fairauction.ScFuncs.StartAuction(ctx.Sign(auctioneer))
	sa.Params.Color().SetValue(tokenColor)
	sa.Params.MinimumBid().SetValue(500)
	sa.Params.Description().SetValue("Cool tokens for sale!")
	transfer := ctx.Transfer()
	transfer.Set(wasmtypes.IOTA, 25) // deposit, must be >=minimum*margin
	transfer.Set(tokenColor, 10)     // the tokens to auction
	sa.Func.Transfer(transfer).Post()
	require.NoError(t, ctx.Err)
	return ctx
}

func TestDeploy(t *testing.T) {
	ctx := wasmsolo.NewSoloContext(t, fairauction.ScName, fairauction.OnLoad)
	require.NoError(t, ctx.ContractExists(fairauction.ScName))
}

func TestFaStartAuction(t *testing.T) {
	ctx := startAuction(t)

	// note 1 iota should be stuck in the delayed finalize_auction
	require.EqualValues(t, 25-1, ctx.Balance(ctx.Account()))
	require.EqualValues(t, 10, ctx.Balance(ctx.Account(), tokenColor))

	// auctioneer sent 25 deposit + 10 tokenColor
	require.EqualValues(t, solo.Saldo-25-10, auctioneer.Balance())
	require.EqualValues(t, 0, auctioneer.Balance(tokenColor))
	require.EqualValues(t, 0, ctx.Balance(auctioneer))

	// remove pending finalize_auction from backlog
	ctx.AdvanceClockBy(61 * time.Minute)
	require.True(t, ctx.WaitForPendingRequests(1))
}

func TestFaAuctionInfo(t *testing.T) {
	ctx := startAuction(t)

	getInfo := fairauction.ScFuncs.GetInfo(ctx)
	getInfo.Params.Color().SetValue(tokenColor)
	getInfo.Func.Call()

	require.NoError(t, ctx.Err)
	require.EqualValues(t, auctioneer.ScAgentID(), getInfo.Results.Creator().Value())
	require.EqualValues(t, 0, getInfo.Results.Bidders().Value())

	// remove pending finalize_auction from backlog
	ctx.AdvanceClockBy(61 * time.Minute)
	require.True(t, ctx.WaitForPendingRequests(1))
}

func TestFaNoBids(t *testing.T) {
	ctx := startAuction(t)

	// wait for finalize_auction
	ctx.AdvanceClockBy(61 * time.Minute)
	require.True(t, ctx.WaitForPendingRequests(1))

	getInfo := fairauction.ScFuncs.GetInfo(ctx)
	getInfo.Params.Color().SetValue(tokenColor)
	getInfo.Func.Call()

	require.NoError(t, ctx.Err)
	require.EqualValues(t, 0, getInfo.Results.Bidders().Value())
}

func TestFaOneBidTooLow(t *testing.T) {
	ctx := startAuction(t)

	bidder := ctx.NewSoloAgent()
	placeBid := fairauction.ScFuncs.PlaceBid(ctx.Sign(bidder))
	placeBid.Params.Color().SetValue(tokenColor)
	placeBid.Func.TransferIotas(100).Post()
	require.Error(t, ctx.Err)

	// TODO this should be a simple 1 request to wait for, but sometimes
	// the finalizeAuction will have already been triggered (bug), so
	// instead of waiting for that single finalizeAuction request we
	// will (erroneously) wait for the inbuf and outbuf counts to equalize
	info := ctx.Chain.MempoolInfo()

	// wait for finalize_auction
	ctx.AdvanceClockBy(61 * time.Minute)
	require.True(t, ctx.WaitForPendingRequests(-info.InBufCounter))

	getInfo := fairauction.ScFuncs.GetInfo(ctx)
	getInfo.Params.Color().SetValue(tokenColor)
	getInfo.Func.Call()

	require.NoError(t, ctx.Err)
	require.EqualValues(t, 0, getInfo.Results.Bidders().Value())
	require.EqualValues(t, 0, getInfo.Results.HighestBid().Value())
}

func TestFaOneBid(t *testing.T) {
	ctx := startAuction(t)

	bidder := ctx.NewSoloAgent()
	placeBid := fairauction.ScFuncs.PlaceBid(ctx.Sign(bidder))
	placeBid.Params.Color().SetValue(tokenColor)
	placeBid.Func.TransferIotas(5000).Post()
	require.NoError(t, ctx.Err)

	// TODO this should be a simple 1 request to wait for, but sometimes
	// the finalizeAuction will have already been triggered (bug), so
	// instead of waiting for that single finalizeAuction request we
	// will (erroneously) wait for the inbuf and outbuf counts to equalize
	info := ctx.Chain.MempoolInfo()

	// wait for finalize_auction
	ctx.AdvanceClockBy(61 * time.Minute)
	require.True(t, ctx.WaitForPendingRequests(-info.InBufCounter))

	getInfo := fairauction.ScFuncs.GetInfo(ctx)
	getInfo.Params.Color().SetValue(tokenColor)
	getInfo.Func.Call()

	require.NoError(t, ctx.Err)
	require.EqualValues(t, 1, getInfo.Results.Bidders().Value())
	require.EqualValues(t, 5000, getInfo.Results.HighestBid().Value())
	require.Equal(t, bidder.ScAddress().AsAgentID(), getInfo.Results.HighestBidder().Value())
}
