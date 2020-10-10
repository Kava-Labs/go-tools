// Code generated by MockGen. DO NOT EDIT.
// Source: bnb_client.go

// Package mock_claim is a generated GoMock package.
package mock_claim

import (
	types "github.com/binance-chain/go-sdk/common/types"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockbnbChainClient is a mock of bnbChainClient interface
type MockbnbChainClient struct {
	ctrl     *gomock.Controller
	recorder *MockbnbChainClientMockRecorder
}

// MockbnbChainClientMockRecorder is the mock recorder for MockbnbChainClient
type MockbnbChainClientMockRecorder struct {
	mock *MockbnbChainClient
}

// NewMockbnbChainClient creates a new mock instance
func NewMockbnbChainClient(ctrl *gomock.Controller) *MockbnbChainClient {
	mock := &MockbnbChainClient{ctrl: ctrl}
	mock.recorder = &MockbnbChainClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockbnbChainClient) EXPECT() *MockbnbChainClientMockRecorder {
	return m.recorder
}

// getSwapByID mocks base method
func (m *MockbnbChainClient) getSwapByID(id types.SwapBytes) (types.AtomicSwap, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getSwapByID", id)
	ret0, _ := ret[0].(types.AtomicSwap)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// getSwapByID indicates an expected call of getSwapByID
func (mr *MockbnbChainClientMockRecorder) getSwapByID(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getSwapByID", reflect.TypeOf((*MockbnbChainClient)(nil).getSwapByID), id)
}
