package app

import (
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/basecoin"
	eyes "github.com/tendermint/merkleeyes/client"
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"

	"github.com/tendermint/basecoin/errors"
	"github.com/tendermint/basecoin/modules/coin"
	"github.com/tendermint/basecoin/stack"
	sm "github.com/tendermint/basecoin/state"
	"github.com/tendermint/basecoin/version"
)

const (
	PluginNameBase = "base"
	ChainKey       = "base/chain_id"
)

type Basecoin struct {
	eyesCli    *eyes.Client
	state      *sm.State
	cacheState *sm.State
	handler    basecoin.Handler
	logger     log.Logger
}

func NewBasecoin(h basecoin.Handler, eyesCli *eyes.Client, l log.Logger) *Basecoin {
	state := sm.NewState(eyesCli, l.With("module", "state"))

	return &Basecoin{
		handler:    h,
		eyesCli:    eyesCli,
		state:      state,
		cacheState: nil,
		logger:     l,
	}
}

// placeholder to just handle sendtx
func DefaultHandler() basecoin.Handler {
	// use the default stack
	h := coin.NewHandler()
	return stack.NewDefault().Use(h)
}

// XXX For testing, not thread safe!
func (app *Basecoin) GetState() *sm.State {
	return app.state.CacheWrap()
}

// ABCI::Info
func (app *Basecoin) Info() abci.ResponseInfo {
	resp, err := app.eyesCli.InfoSync()
	if err != nil {
		cmn.PanicCrisis(err)
	}
	return abci.ResponseInfo{
		Data:             cmn.Fmt("Basecoin v%v", version.Version),
		LastBlockHeight:  resp.LastBlockHeight,
		LastBlockAppHash: resp.LastBlockAppHash,
	}
}

// ABCI::SetOption
func (app *Basecoin) SetOption(key string, value string) string {
	if key == ChainKey {
		app.state.SetChainID(value)
		return "Success"
	}

	log, err := app.handler.SetOption(app.logger, app.state, key, value)
	if err == nil {
		return log
	}
	return "Error: " + err.Error()
}

// ABCI::DeliverTx
func (app *Basecoin) DeliverTx(txBytes []byte) abci.Result {
	tx, err := basecoin.LoadTx(txBytes)
	if err != nil {
		return errors.Result(err)
	}

	// TODO: can we abstract this setup and commit logic??
	cache := app.state.CacheWrap()
	ctx := stack.NewContext(
		app.state.GetChainID(),
		app.logger.With("call", "delivertx"),
	)
	res, err := app.handler.DeliverTx(ctx, cache, tx)

	if err != nil {
		// discard the cache...
		return errors.Result(err)
	}
	// commit the cache and return result
	cache.CacheSync()
	return res.ToABCI()
}

// ABCI::CheckTx
func (app *Basecoin) CheckTx(txBytes []byte) abci.Result {
	tx, err := basecoin.LoadTx(txBytes)
	if err != nil {
		return errors.Result(err)
	}

	// TODO: can we abstract this setup and commit logic??
	ctx := stack.NewContext(
		app.state.GetChainID(),
		app.logger.With("call", "checktx"),
	)
	// checktx generally shouldn't touch the state, but we don't care
	// here on the framework level, since the cacheState is thrown away next block
	res, err := app.handler.CheckTx(ctx, app.cacheState, tx)

	if err != nil {
		return errors.Result(err)
	}
	return res.ToABCI()
}

// ABCI::Query
func (app *Basecoin) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {
	if len(reqQuery.Data) == 0 {
		resQuery.Log = "Query cannot be zero length"
		resQuery.Code = abci.CodeType_EncodingError
		return
	}

	resQuery, err := app.eyesCli.QuerySync(reqQuery)
	if err != nil {
		resQuery.Log = "Failed to query MerkleEyes: " + err.Error()
		resQuery.Code = abci.CodeType_InternalError
		return
	}
	return
}

// ABCI::Commit
func (app *Basecoin) Commit() (res abci.Result) {

	// Commit state
	res = app.state.Commit()

	// Wrap the committed state in cache for CheckTx
	app.cacheState = app.state.CacheWrap()

	if res.IsErr() {
		cmn.PanicSanity("Error getting hash: " + res.Error())
	}
	return res
}

// ABCI::InitChain
func (app *Basecoin) InitChain(validators []*abci.Validator) {
	// for _, plugin := range app.plugins.GetList() {
	// 	plugin.InitChain(app.state, validators)
	// }
}

// ABCI::BeginBlock
func (app *Basecoin) BeginBlock(hash []byte, header *abci.Header) {
	// for _, plugin := range app.plugins.GetList() {
	// 	plugin.BeginBlock(app.state, hash, header)
	// }
}

// ABCI::EndBlock
func (app *Basecoin) EndBlock(height uint64) (res abci.ResponseEndBlock) {
	// for _, plugin := range app.plugins.GetList() {
	// 	pluginRes := plugin.EndBlock(app.state, height)
	// 	res.Diffs = append(res.Diffs, pluginRes.Diffs...)
	// }
	return
}
