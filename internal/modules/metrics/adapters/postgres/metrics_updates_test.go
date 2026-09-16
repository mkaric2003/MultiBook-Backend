package postgres

import (
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

func TestDashboardUpdateSubscriptionsAreScopedAndCoalesced(t *testing.T) {
	listener := NewMetricsUpdateListener("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	firstBusiness, secondBusiness := uuid.New(), uuid.New()
	first, cancelFirst := listener.SubscribeChanges(firstBusiness)
	defer cancelFirst()
	second, cancelSecond := listener.SubscribeChanges(secondBusiness)
	defer cancelSecond()

	listener.publish(firstBusiness)
	listener.publish(firstBusiness)
	select {
	case <-first:
	default:
		t.Fatal("first business did not receive an update")
	}
	select {
	case <-first:
		t.Fatal("duplicate pending updates were not coalesced")
	default:
	}
	select {
	case <-second:
		t.Fatal("second business received another business's update")
	default:
	}

	cancelFirst()
	listener.publish(firstBusiness)
	select {
	case <-first:
		t.Fatal("cancelled subscription received an update")
	default:
	}
}
