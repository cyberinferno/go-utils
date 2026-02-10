package perfmonitor

import "time"

// PerformanceMonitor helps track elapsed time between operations
type PerformanceMonitor struct {
	startTime time.Time
	endTime   time.Time
}

// Start begins the performance monitoring
func (p *PerformanceMonitor) Start() {
	p.startTime = time.Now()
}

// Stop ends the performance monitoring
func (p *PerformanceMonitor) Stop() {
	if p.startTime.IsZero() {
		return
	}

	p.endTime = time.Now()
}

// ElapsedMilliseconds returns the elapsed time in milliseconds between Start and Stop
func (p *PerformanceMonitor) ElapsedMilliseconds() float64 {
	if p.startTime.IsZero() || p.endTime.IsZero() {
		return 0
	}

	return float64(p.endTime.Sub(p.startTime).Microseconds()) / 1000.0
}

// Reset clears the timer values to allow reuse
func (p *PerformanceMonitor) Reset() {
	p.startTime = time.Time{}
	p.endTime = time.Time{}
}

// NewPerformanceMonitor creates a new PerformanceMonitor instance
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{}
}
