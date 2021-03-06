package client

import (
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/23-commitment/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
)

// QueryTendermintProof performs an ABCI query with the given key and returns
// the value of the query, the proto encoded merkle proof, and the height of
// the Tendermint block containing the state root. The desired tendermint height
// to perform the query should be set in the client context. The query will be
// performed at one below this height (at the IAVL version) in order to obtain
// the correct merkle proof. Proof queries at height less than or equal to 2 are
// not supported.
// Issue: https://github.com/cosmos/cosmos-sdk/issues/6567
func QueryTendermintProof(clientCtx client.Context, key []byte) ([]byte, []byte, clienttypes.Height, error) {
	height := clientCtx.Height

	// ABCI queries at height less than or equal to 2 are not supported.
	// Base app does not support queries for height less than or equal to 1.
	// Therefore, a query at height 2 would be equivalent to a query at height 3
	if clientCtx.Height <= 2 {
		return nil, nil, clienttypes.Height{}, fmt.Errorf("proof queries at height <= 2 are not supported")
	}

	// Use the IAVL height if a valid tendermint height is passed in.
	height--

	req := abci.RequestQuery{
		Path:   fmt.Sprintf("store/%s/key", host.StoreKey),
		Height: height,
		Data:   key,
		Prove:  true,
	}

	res, err := clientCtx.QueryABCI(req)
	if err != nil {
		return nil, nil, clienttypes.Height{}, err
	}

	merkleProof := commitmenttypes.MerkleProof{
		Proof: res.ProofOps,
	}

	cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)

	proofBz, err := cdc.MarshalBinaryBare(&merkleProof)
	if err != nil {
		return nil, nil, clienttypes.Height{}, err
	}

	version := clienttypes.ParseChainID(clientCtx.ChainID)
	return res.Value, proofBz, clienttypes.NewHeight(version, uint64(res.Height)+1), nil
}
