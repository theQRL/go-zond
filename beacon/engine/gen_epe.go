// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package engine

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/theQRL/go-zond/common/hexutil"
)

var _ = (*executionPayloadEnvelopeMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (e ExecutionPayloadEnvelope) MarshalJSON() ([]byte, error) {
	type ExecutionPayloadEnvelope struct {
		ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
		BlockValue       *hexutil.Big    `json:"blockValue"  gencodec:"required"`
		Override         bool            `json:"shouldOverrideBuilder"`
	}
	var enc ExecutionPayloadEnvelope
	enc.ExecutionPayload = e.ExecutionPayload
	enc.BlockValue = (*hexutil.Big)(e.BlockValue)
	enc.Override = e.Override
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (e *ExecutionPayloadEnvelope) UnmarshalJSON(input []byte) error {
	type ExecutionPayloadEnvelope struct {
		ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
		BlockValue       *hexutil.Big    `json:"blockValue"  gencodec:"required"`
		Override         *bool           `json:"shouldOverrideBuilder"`
	}
	var dec ExecutionPayloadEnvelope
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ExecutionPayload == nil {
		return errors.New("missing required field 'executionPayload' for ExecutionPayloadEnvelope")
	}
	e.ExecutionPayload = dec.ExecutionPayload
	if dec.BlockValue == nil {
		return errors.New("missing required field 'blockValue' for ExecutionPayloadEnvelope")
	}
	e.BlockValue = (*big.Int)(dec.BlockValue)
	if dec.Override != nil {
		e.Override = *dec.Override
	}
	return nil
}
