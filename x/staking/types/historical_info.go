package types

import (
	"sort"

	"github.com/tendermint/tendermint/crypto"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewHistoricalInfo will create a historical information struct from header and valset
// it will first sort valset before inclusion into historical info
func NewHistoricalInfo(header tmproto.Header, valSet Validators) HistoricalInfo {
	sort.Sort(valSet)

	return HistoricalInfo{
		Header: header,
		Valset: valSet,
	}
}

// MustMarshalHistoricalInfo wll marshal historical info and panic on error
func MustMarshalHistoricalInfo(cdc codec.BinaryMarshaler, hi *HistoricalInfo) []byte {
	return cdc.MustMarshalBinaryBare(hi)
}

// MustUnmarshalHistoricalInfo wll unmarshal historical info and panic on error
func MustUnmarshalHistoricalInfo(cdc codec.BinaryMarshaler, value []byte) HistoricalInfo {
	hi, err := UnmarshalHistoricalInfo(cdc, value)
	if err != nil {
		panic(err)
	}

	return hi
}

// UnmarshalHistoricalInfo will unmarshal historical info and return any error
func UnmarshalHistoricalInfo(cdc codec.BinaryMarshaler, value []byte) (hi HistoricalInfo, err error) {
	err = cdc.UnmarshalBinaryBare(value, &hi)

	return hi, err
}

// ValidateBasic will ensure HistoricalInfo is not nil and sorted
func ValidateBasic(hi HistoricalInfo) error {
	if len(hi.Valset) == 0 {
		return sdkerrors.Wrap(ErrInvalidHistoricalInfo, "validator set is empty")
	}

	if !sort.IsSorted(Validators(hi.Valset)) {
		return sdkerrors.Wrap(ErrInvalidHistoricalInfo, "validator set is not sorted by address")
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (hi *HistoricalInfo) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	// TODO: do we need to check all validators in Valset?
	for i := range hi.Valset {
		var pk crypto.PubKey
		if err := unpacker.UnpackAny(hi.Valset[i].ConsensusPubkey, &pk); err != nil {
			return err
		}
	}
	return nil
}
