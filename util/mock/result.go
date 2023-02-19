// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"github.com/machbase/neo-spi"
	"sync"
)

// Ensure, that ResultMock does implement spi.Result.
// If this is not the case, regenerate this file with moq.
var _ spi.Result = &ResultMock{}

// ResultMock is a mock implementation of spi.Result.
//
//	func TestSomethingThatUsesResult(t *testing.T) {
//
//		// make and configure a mocked spi.Result
//		mockedResult := &ResultMock{
//			ErrFunc: func() error {
//				panic("mock out the Err method")
//			},
//			MessageFunc: func() string {
//				panic("mock out the Message method")
//			},
//			RowsAffectedFunc: func() int64 {
//				panic("mock out the RowsAffected method")
//			},
//		}
//
//		// use mockedResult in code that requires spi.Result
//		// and then make assertions.
//
//	}
type ResultMock struct {
	// ErrFunc mocks the Err method.
	ErrFunc func() error

	// MessageFunc mocks the Message method.
	MessageFunc func() string

	// RowsAffectedFunc mocks the RowsAffected method.
	RowsAffectedFunc func() int64

	// calls tracks calls to the methods.
	calls struct {
		// Err holds details about calls to the Err method.
		Err []struct {
		}
		// Message holds details about calls to the Message method.
		Message []struct {
		}
		// RowsAffected holds details about calls to the RowsAffected method.
		RowsAffected []struct {
		}
	}
	lockErr          sync.RWMutex
	lockMessage      sync.RWMutex
	lockRowsAffected sync.RWMutex
}

// Err calls ErrFunc.
func (mock *ResultMock) Err() error {
	if mock.ErrFunc == nil {
		panic("ResultMock.ErrFunc: method is nil but Result.Err was just called")
	}
	callInfo := struct {
	}{}
	mock.lockErr.Lock()
	mock.calls.Err = append(mock.calls.Err, callInfo)
	mock.lockErr.Unlock()
	return mock.ErrFunc()
}

// ErrCalls gets all the calls that were made to Err.
// Check the length with:
//
//	len(mockedResult.ErrCalls())
func (mock *ResultMock) ErrCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockErr.RLock()
	calls = mock.calls.Err
	mock.lockErr.RUnlock()
	return calls
}

// Message calls MessageFunc.
func (mock *ResultMock) Message() string {
	if mock.MessageFunc == nil {
		panic("ResultMock.MessageFunc: method is nil but Result.Message was just called")
	}
	callInfo := struct {
	}{}
	mock.lockMessage.Lock()
	mock.calls.Message = append(mock.calls.Message, callInfo)
	mock.lockMessage.Unlock()
	return mock.MessageFunc()
}

// MessageCalls gets all the calls that were made to Message.
// Check the length with:
//
//	len(mockedResult.MessageCalls())
func (mock *ResultMock) MessageCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockMessage.RLock()
	calls = mock.calls.Message
	mock.lockMessage.RUnlock()
	return calls
}

// RowsAffected calls RowsAffectedFunc.
func (mock *ResultMock) RowsAffected() int64 {
	if mock.RowsAffectedFunc == nil {
		panic("ResultMock.RowsAffectedFunc: method is nil but Result.RowsAffected was just called")
	}
	callInfo := struct {
	}{}
	mock.lockRowsAffected.Lock()
	mock.calls.RowsAffected = append(mock.calls.RowsAffected, callInfo)
	mock.lockRowsAffected.Unlock()
	return mock.RowsAffectedFunc()
}

// RowsAffectedCalls gets all the calls that were made to RowsAffected.
// Check the length with:
//
//	len(mockedResult.RowsAffectedCalls())
func (mock *ResultMock) RowsAffectedCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockRowsAffected.RLock()
	calls = mock.calls.RowsAffected
	mock.lockRowsAffected.RUnlock()
	return calls
}
