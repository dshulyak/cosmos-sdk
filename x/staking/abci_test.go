package staking

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestBeginBlocker(t *testing.T) {
	ctx, _, k, _ := keeper.CreateTestInput(t, false, 10)

	// set historical entries in params to 5
	params := types.DefaultParams()
	params.HistoricalEntries = 5
	k.SetParams(ctx, params)

	// set historical info at 5 which should be pruned
	header := abci.Header{
		ChainID: "HelloChain",
		Height:  5,
	}
	valSet := []Validator{
		types.NewValidator(sdk.ValAddress(keeper.Addrs[0]), keeper.PKs[0], types.Description{}),
		types.NewValidator(sdk.ValAddress(keeper.Addrs[1]), keeper.PKs[1], types.Description{}),
	}
	hi5 := types.NewHistoricalInfo(header, valSet)
	k.SetHistoricalInfo(ctx, 5, hi5)
	recv, found := k.GetHistoricalInfo(ctx, 5)
	require.True(t, found)
	require.Equal(t, hi5, recv)

	// Set last validators in keeper
	val1 := types.NewValidator(sdk.ValAddress(keeper.Addrs[2]), keeper.PKs[2], types.Description{})
	k.SetValidator(ctx, val1)
	k.SetLastValidatorPower(ctx, val1.OperatorAddress, 10)
	val2 := types.NewValidator(sdk.ValAddress(keeper.Addrs[3]), keeper.PKs[3], types.Description{})
	vals := []types.Validator{val1, val2}
	sort.Sort(types.Validators(vals))
	k.SetValidator(ctx, val2)
	k.SetLastValidatorPower(ctx, val2.OperatorAddress, 8)

	// Set Header for BeginBlock context
	header = abci.Header{
		ChainID: "HelloChain",
		Height:  10,
	}
	ctx = ctx.WithBlockHeader(header)

	BeginBlocker(ctx, k)

	// Check HistoricalInfo at height 10 is persisted
	expected := types.HistoricalInfo{
		Header: header,
		ValSet: vals,
	}
	recv, found = k.GetHistoricalInfo(ctx, 10)
	require.True(t, found, "GetHistoricalInfo failed after BeginBlock")
	require.Equal(t, expected, recv, "GetHistoricalInfo returned eunexpected result")

	// Check HistoricalInfo at height 5 is pruned
	recv, found = k.GetHistoricalInfo(ctx, 5)
	require.False(t, found, "GetHistoricalInfo did not prune")
	require.Equal(t, types.HistoricalInfo{}, recv, "GetHistoricalInfo is not empty after prune")
}