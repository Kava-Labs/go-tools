// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/kava-labs/go-tools/deputy-claimer/claim (interfaces: BnbChainClient)

// Package mock is a generated GoMock package.
package mock

import (
	rpc "github.com/binance-chain/go-sdk/client/rpc"
	types "github.com/binance-chain/go-sdk/common/types"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockBnbChainClient is a mock of BnbChainClient interface
type MockBnbChainClient struct {
	ctrl     *gomock.Controller
	recorder *MockBnbChainClientMockRecorder
}

// MockBnbChainClientMockRecorder is the mock recorder for MockBnbChainClient
type MockBnbChainClientMockRecorder struct {
	mock *MockBnbChainClient
}

// NewMockBnbChainClient creates a new mock instance
func NewMockBnbChainClient(ctrl *gomock.Controller) *MockBnbChainClient {
	mock := &MockBnbChainClient{ctrl: ctrl}
	mock.recorder = &MockBnbChainClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockBnbChainClient) EXPECT() *MockBnbChainClientMockRecorder {
	return m.recorder
}

// GetBNBSDKClient mocks base method
func (m *MockBnbChainClient) GetBNBSDKClient() *rpc.HTTP {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBNBSDKClient")
	ret0, _ := ret[0].(*rpc.HTTP)
	return ret0
}

// GetBNBSDKClient indicates an expected call of GetBNBSDKClient
func (mr *MockBnbChainClientMockRecorder) GetBNBSDKClient() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBNBSDKClient", reflect.TypeOf((*MockBnbChainClient)(nil).GetBNBSDKClient))
}

// GetOpenOutgoingSwaps mocks base method
func (m *MockBnbChainClient) GetOpenOutgoingSwaps() ([]types.AtomicSwap, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOpenOutgoingSwaps")
	ret0, _ := ret[0].([]types.AtomicSwap)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOpenOutgoingSwaps indicates an expected call of GetOpenOutgoingSwaps
func (mr *MockBnbChainClientMockRecorder) GetOpenOutgoingSwaps() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOpenOutgoingSwaps", reflect.TypeOf((*MockBnbChainClient)(nil).GetOpenOutgoingSwaps))
}

// GetRandomNumberFromSwap mocks base method
func (m *MockBnbChainClient) GetRandomNumberFromSwap(arg0 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRandomNumberFromSwap", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRandomNumberFromSwap indicates an expected call of GetRandomNumberFromSwap
func (mr *MockBnbChainClientMockRecorder) GetRandomNumberFromSwap(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRandomNumberFromSwap", reflect.TypeOf((*MockBnbChainClient)(nil).GetRandomNumberFromSwap), arg0)
}

// GetTxConfirmation mocks base method
func (m *MockBnbChainClient) GetTxConfirmation(arg0 []byte) (*rpc.ResultTx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTxConfirmation", arg0)
	ret0, _ := ret[0].(*rpc.ResultTx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTxConfirmation indicates an expected call of GetTxConfirmation
func (mr *MockBnbChainClientMockRecorder) GetTxConfirmation(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTxConfirmation", reflect.TypeOf((*MockBnbChainClient)(nil).GetTxConfirmation), arg0)
}
