// SWE100821: SensorSource interface — abstraction for any sensor data provider.
// Implementations: phone (ADB), Xavier (sysfs thermals), I2C devices, GPIO pins.
// Each source discovers itself, polls readings, and reports failures gracefully.

package sensors

import (
	"context"
	"time"
)

// SensorSource provides readings from a category of sensors.
// Sources are discovered at startup and polled continuously.
type SensorSource interface {
	// Name returns a human-readable source identifier (e.g. "phone", "xavier", "i2c-bus-1").
	Name() string

	// Available returns true if this source can currently provide readings.
	// Called during discovery and periodically to detect hot-plug.
	Available() bool

	// Poll reads all sensors from this source. Returns partial results on partial failure.
	// Must not block longer than the context deadline.
	Poll(ctx context.Context) ([]SensorReading, error)

	// Interval returns the recommended polling interval for this source.
	Interval() time.Duration
}
