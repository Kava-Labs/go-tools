package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func TestParseABCIResult(t *testing.T) {
	mockOKResponse := &ctypes.ResultABCIQuery{
		Response: abci.ResponseQuery{
			Code:  uint32(0),
			Log:   "",
			Value: []byte("{}"),
		},
	}

	mockNotOKResponse := &ctypes.ResultABCIQuery{
		Response: abci.ResponseQuery{
			Code:  uint32(1),
			Log:   "internal error",
			Value: []byte("{}"),
		},
	}

	mockNilByteResponse := &ctypes.ResultABCIQuery{
		Response: abci.ResponseQuery{
			Code:  uint32(0),
			Log:   "",
			Value: []byte(nil),
		},
	}

	mockABCIError := errors.New("abci error")

	// if abci errors, we return error and empty bytes
	data, err := ParseABCIResult(mockOKResponse, mockABCIError)
	assert.Equal(t, []byte{}, data)
	assert.Equal(t, mockABCIError, err)

	// if response is not OK, we return log error with empty bytes
	data, err = ParseABCIResult(mockNotOKResponse, nil)
	assert.Equal(t, []byte{}, data)
	assert.Equal(t, errors.New(mockNotOKResponse.Response.Log), err)

	// if response is OK , we return nil error with Reponse value
	data, err = ParseABCIResult(mockOKResponse, nil)
	assert.Equal(t, mockOKResponse.Response.Value, data)
	assert.Nil(t, err)

	// if response is len 0, we return
	data, err = ParseABCIResult(mockNilByteResponse, nil)
	assert.Equal(t, []byte{}, data)
	assert.Nil(t, err)
}
