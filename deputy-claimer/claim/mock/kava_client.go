// Code generated by MockGen. DO NOT EDIT.
// Source: kava_client.go

// Package mock_claim is a generated GoMock package.
package mock_claim

import (
	gomock "github.com/golang/mock/gomock"
	codec "github.com/kava-labs/cosmos-sdk/codec"
	types "github.com/kava-labs/cosmos-sdk/types"
	exported "github.com/kava-labs/cosmos-sdk/x/auth/exported"
	bep3 "github.com/kava-labs/go-sdk/kava/bep3"
	coretypes "github.com/kava-labs/tendermint/rpc/core/types"
	types0 "github.com/kava-labs/tendermint/types"
	reflect "reflect"
)

// MockkavaChainClient is a mock of kavaChainClient interface
type MockkavaChainClient struct {
	ctrl     *gomock.Controller
	recorder *MockkavaChainClientMockRecorder
}

// MockkavaChainClientMockRecorder is the mock recorder for MockkavaChainClient
type MockkavaChainClientMockRecorder struct {
	mock *MockkavaChainClient
}

// NewMockkavaChainClient creates a new mock instance
func NewMockkavaChainClient(ctrl *gomock.Controller) *MockkavaChainClient {
	mock := &MockkavaChainClient{ctrl: ctrl}
	mock.recorder = &MockkavaChainClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockkavaChainClient) EXPECT() *MockkavaChainClientMockRecorder {
	return m.recorder
}

// getOpenSwaps mocks base method
func (m *MockkavaChainClient) getOpenSwaps() (bep3.AtomicSwaps, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getOpenSwaps")
	ret0, _ := ret[0].(bep3.AtomicSwaps)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// getOpenSwaps indicates an expected call of getOpenSwaps
func (mr *MockkavaChainClientMockRecorder) getOpenSwaps() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getOpenSwaps", reflect.TypeOf((*MockkavaChainClient)(nil).getOpenSwaps))
}

// getAccount mocks base method
func (m *MockkavaChainClient) getAccount(address types.AccAddress) (exported.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getAccount", address)
	ret0, _ := ret[0].(exported.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// getAccount indicates an expected call of getAccount
func (mr *MockkavaChainClientMockRecorder) getAccount(address interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getAccount", reflect.TypeOf((*MockkavaChainClient)(nil).getAccount), address)
}

// getTxConfirmation mocks base method
func (m *MockkavaChainClient) getTxConfirmation(txHash []byte) (*coretypes.ResultTx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getTxConfirmation", txHash)
	ret0, _ := ret[0].(*coretypes.ResultTx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// getTxConfirmation indicates an expected call of getTxConfirmation
func (mr *MockkavaChainClientMockRecorder) getTxConfirmation(txHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getTxConfirmation", reflect.TypeOf((*MockkavaChainClient)(nil).getTxConfirmation), txHash)
}

// broadcastTx mocks base method
func (m *MockkavaChainClient) broadcastTx(tx types0.Tx) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "broadcastTx", tx)
	ret0, _ := ret[0].(error)
	return ret0
}

// broadcastTx indicates an expected call of broadcastTx
func (mr *MockkavaChainClientMockRecorder) broadcastTx(tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "broadcastTx", reflect.TypeOf((*MockkavaChainClient)(nil).broadcastTx), tx)
}

// getChainID mocks base method
func (m *MockkavaChainClient) getChainID() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getChainID")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// getChainID indicates an expected call of getChainID
func (mr *MockkavaChainClientMockRecorder) getChainID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getChainID", reflect.TypeOf((*MockkavaChainClient)(nil).getChainID))
}

// getCodec mocks base method
func (m *MockkavaChainClient) getCodec() *codec.Codec {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getCodec")
	ret0, _ := ret[0].(*codec.Codec)
	return ret0
}

// getCodec indicates an expected call of getCodec
func (mr *MockkavaChainClientMockRecorder) getCodec() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getCodec", reflect.TypeOf((*MockkavaChainClient)(nil).getCodec))
}
