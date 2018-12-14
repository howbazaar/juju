// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/jujuclient (interfaces: CredentialStore)

// Package common is a generated GoMock package.
package common

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	cloud "github.com/juju/juju/cloud"
)

// MockCredentialStore is a mock of CredentialStore interface
type MockCredentialStore struct {
	ctrl     *gomock.Controller
	recorder *MockCredentialStoreMockRecorder
}

// MockCredentialStoreMockRecorder is the mock recorder for MockCredentialStore
type MockCredentialStoreMockRecorder struct {
	mock *MockCredentialStore
}

// NewMockCredentialStore creates a new mock instance
func NewMockCredentialStore(ctrl *gomock.Controller) *MockCredentialStore {
	mock := &MockCredentialStore{ctrl: ctrl}
	mock.recorder = &MockCredentialStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCredentialStore) EXPECT() *MockCredentialStoreMockRecorder {
	return m.recorder
}

// AllCredentials mocks base method
func (m *MockCredentialStore) AllCredentials() (map[string]cloud.CloudCredential, error) {
	ret := m.ctrl.Call(m, "AllCredentials")
	ret0, _ := ret[0].(map[string]cloud.CloudCredential)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AllCredentials indicates an expected call of AllCredentials
func (mr *MockCredentialStoreMockRecorder) AllCredentials() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllCredentials", reflect.TypeOf((*MockCredentialStore)(nil).AllCredentials))
}

// CredentialForCloud mocks base method
func (m *MockCredentialStore) CredentialForCloud(arg0 string) (*cloud.CloudCredential, error) {
	ret := m.ctrl.Call(m, "CredentialForCloud", arg0)
	ret0, _ := ret[0].(*cloud.CloudCredential)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CredentialForCloud indicates an expected call of CredentialForCloud
func (mr *MockCredentialStoreMockRecorder) CredentialForCloud(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CredentialForCloud", reflect.TypeOf((*MockCredentialStore)(nil).CredentialForCloud), arg0)
}

// UpdateCredential mocks base method
func (m *MockCredentialStore) UpdateCredential(arg0 string, arg1 cloud.CloudCredential) error {
	ret := m.ctrl.Call(m, "UpdateCredential", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateCredential indicates an expected call of UpdateCredential
func (mr *MockCredentialStoreMockRecorder) UpdateCredential(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCredential", reflect.TypeOf((*MockCredentialStore)(nil).UpdateCredential), arg0, arg1)
}
