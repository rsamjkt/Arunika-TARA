package antrol

import (
	"context"
	"sync"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// MockAntrolClient adalah implementasi AntrolClient untuk testing.
type MockAntrolClient struct {
	GetBookingHariIniFunc func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error)
	PushAntrianFunc       func(ctx context.Context, req domain.AntrianRequest, ticket *domain.Ticket) error

	mu        sync.Mutex
	callCount map[string]int
}

var _ AntrolClient = (*MockAntrolClient)(nil)

func NewMock() *MockAntrolClient {
	return &MockAntrolClient{callCount: make(map[string]int)}
}

func (m *MockAntrolClient) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount[method]
}

func (m *MockAntrolClient) recordCall(name string) {
	m.mu.Lock()
	m.callCount[name]++
	m.mu.Unlock()
}

func (m *MockAntrolClient) GetBookingHariIni(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
	m.recordCall("GetBookingHariIni")
	if m.GetBookingHariIniFunc != nil {
		return m.GetBookingHariIniFunc(ctx, noKartu, tgl)
	}
	return nil, nil
}

func (m *MockAntrolClient) PushAntrian(ctx context.Context, req domain.AntrianRequest, ticket *domain.Ticket) error {
	m.recordCall("PushAntrian")
	if m.PushAntrianFunc != nil {
		return m.PushAntrianFunc(ctx, req, ticket)
	}
	return nil
}
