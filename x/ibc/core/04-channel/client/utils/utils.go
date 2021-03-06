package utils

import (
	"context"
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clientutils "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/client/utils"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	ibcclient "github.com/cosmos/cosmos-sdk/x/ibc/core/client"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
)

// QueryPacketCommitment returns a packet commitment.
// If prove is true, it performs an ABCI store query in order to retrieve the merkle proof. Otherwise,
// it uses the gRPC query client.
func QueryPacketCommitment(
	clientCtx client.Context, portID, channelID string,
	sequence uint64, prove bool,
) (*types.QueryPacketCommitmentResponse, error) {
	if prove {
		return queryPacketCommitmentABCI(clientCtx, portID, channelID, sequence)
	}

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryPacketCommitmentRequest{
		PortId:    portID,
		ChannelId: channelID,
		Sequence:  sequence,
	}

	return queryClient.PacketCommitment(context.Background(), req)
}

func queryPacketCommitmentABCI(
	clientCtx client.Context, portID, channelID string, sequence uint64,
) (*types.QueryPacketCommitmentResponse, error) {
	key := host.KeyPacketCommitment(portID, channelID, sequence)

	value, proofBz, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, key)
	if err != nil {
		return nil, err
	}

	// check if packet commitment exists
	if len(value) == 0 {
		return nil, sdkerrors.Wrapf(types.ErrPacketCommitmentNotFound, "portID (%s), channelID (%s), sequence (%d)", portID, channelID, sequence)
	}

	return types.NewQueryPacketCommitmentResponse(portID, channelID, sequence, value, proofBz, proofHeight), nil
}

// QueryChannel returns a channel end.
// If prove is true, it performs an ABCI store query in order to retrieve the merkle proof. Otherwise,
// it uses the gRPC query client.
func QueryChannel(
	clientCtx client.Context, portID, channelID string, prove bool,
) (*types.QueryChannelResponse, error) {
	if prove {
		return queryChannelABCI(clientCtx, portID, channelID)
	}

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	}

	return queryClient.Channel(context.Background(), req)
}

func queryChannelABCI(clientCtx client.Context, portID, channelID string) (*types.QueryChannelResponse, error) {
	key := host.KeyChannel(portID, channelID)

	value, proofBz, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, key)
	if err != nil {
		return nil, err
	}

	// check if channel exists
	if len(value) == 0 {
		return nil, sdkerrors.Wrapf(types.ErrChannelNotFound, "portID (%s), channelID (%s)", portID, channelID)
	}

	cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)

	var channel types.Channel
	if err := cdc.UnmarshalBinaryBare(value, &channel); err != nil {
		return nil, err
	}

	return types.NewQueryChannelResponse(portID, channelID, channel, proofBz, proofHeight), nil
}

// QueryChannelClientState returns the ClientState of a channel end. If
// prove is true, it performs an ABCI store query in order to retrieve the
// merkle proof. Otherwise, it uses the gRPC query client.
func QueryChannelClientState(
	clientCtx client.Context, portID, channelID string, prove bool,
) (*types.QueryChannelClientStateResponse, error) {

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryChannelClientStateRequest{
		PortId:    portID,
		ChannelId: channelID,
	}

	res, err := queryClient.ChannelClientState(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if prove {
		clientStateRes, err := clientutils.QueryClientStateABCI(clientCtx, res.IdentifiedClientState.ClientId)
		if err != nil {
			return nil, err
		}

		// use client state returned from ABCI query in case query height differs
		identifiedClientState := clienttypes.IdentifiedClientState{
			ClientId:    res.IdentifiedClientState.ClientId,
			ClientState: clientStateRes.ClientState,
		}
		res = types.NewQueryChannelClientStateResponse(identifiedClientState, clientStateRes.Proof, clientStateRes.ProofHeight)
	}

	return res, nil
}

// QueryChannelConsensusState returns the ConsensusState of a channel end. If
// prove is true, it performs an ABCI store query in order to retrieve the
// merkle proof. Otherwise, it uses the gRPC query client.
func QueryChannelConsensusState(
	clientCtx client.Context, portID, channelID string, height clienttypes.Height, prove bool,
) (*types.QueryChannelConsensusStateResponse, error) {

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryChannelConsensusStateRequest{
		PortId:        portID,
		ChannelId:     channelID,
		VersionNumber: height.VersionNumber,
		VersionHeight: height.VersionHeight,
	}

	res, err := queryClient.ChannelConsensusState(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if prove {
		consensusStateRes, err := clientutils.QueryConsensusStateABCI(clientCtx, res.ClientId, height)
		if err != nil {
			return nil, err
		}

		res = types.NewQueryChannelConsensusStateResponse(res.ClientId, consensusStateRes.ConsensusState, height, consensusStateRes.Proof, consensusStateRes.ProofHeight)
	}

	return res, nil
}

// QueryLatestConsensusState uses the channel Querier to return the
// latest ConsensusState given the source port ID and source channel ID.
func QueryLatestConsensusState(
	clientCtx client.Context, portID, channelID string,
) (exported.ConsensusState, clienttypes.Height, clienttypes.Height, error) {
	clientRes, err := QueryChannelClientState(clientCtx, portID, channelID, false)
	if err != nil {
		return nil, clienttypes.Height{}, clienttypes.Height{}, err
	}
	clientState, err := clienttypes.UnpackClientState(clientRes.IdentifiedClientState.ClientState)
	if err != nil {
		return nil, clienttypes.Height{}, clienttypes.Height{}, err
	}

	clientHeight, ok := clientState.GetLatestHeight().(clienttypes.Height)
	if !ok {
		return nil, clienttypes.Height{}, clienttypes.Height{}, sdkerrors.Wrapf(sdkerrors.ErrInvalidHeight, "invalid height type. expected type: %T, got: %T",
			clienttypes.Height{}, clientHeight)
	}
	res, err := QueryChannelConsensusState(clientCtx, portID, channelID, clientHeight, false)
	if err != nil {
		return nil, clienttypes.Height{}, clienttypes.Height{}, err
	}

	consensusState, err := clienttypes.UnpackConsensusState(res.ConsensusState)
	if err != nil {
		return nil, clienttypes.Height{}, clienttypes.Height{}, err
	}

	return consensusState, clientHeight, res.ProofHeight, nil
}

// QueryNextSequenceReceive returns the next sequence receive.
// If prove is true, it performs an ABCI store query in order to retrieve the merkle proof. Otherwise,
// it uses the gRPC query client.
func QueryNextSequenceReceive(
	clientCtx client.Context, portID, channelID string, prove bool,
) (*types.QueryNextSequenceReceiveResponse, error) {
	if prove {
		return queryNextSequenceRecvABCI(clientCtx, portID, channelID)
	}

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryNextSequenceReceiveRequest{
		PortId:    portID,
		ChannelId: channelID,
	}

	return queryClient.NextSequenceReceive(context.Background(), req)
}

func queryNextSequenceRecvABCI(clientCtx client.Context, portID, channelID string) (*types.QueryNextSequenceReceiveResponse, error) {
	key := host.KeyNextSequenceRecv(portID, channelID)

	value, proofBz, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, key)
	if err != nil {
		return nil, err
	}

	// check if next sequence receive exists
	if len(value) == 0 {
		return nil, sdkerrors.Wrapf(types.ErrChannelNotFound, "portID (%s), channelID (%s)", portID, channelID)
	}

	sequence := binary.BigEndian.Uint64(value)

	return types.NewQueryNextSequenceReceiveResponse(portID, channelID, sequence, proofBz, proofHeight), nil
}
