package maintenance

import "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"

type validationIOTracker struct {
	last        uint64
	accumulated uint64
	available   bool
	sampled     bool
}

type validationEstimate struct {
	Available  bool
	BytesDone  uint64
	BytesTotal uint64
	Progress   int
}

func newValidationIOTracker() validationIOTracker {
	value, ok := steam.ClientReadBytes()
	return validationIOTracker{last: value, available: ok, sampled: ok}
}

func (t *validationIOTracker) observe(active bool) uint64 {
	value, ok := steam.ClientReadBytes()
	return t.observeValue(active, value, ok)
}

func (t *validationIOTracker) observeValue(active bool, value uint64, ok bool) uint64 {
	if !ok {
		return 0
	}
	if !t.sampled || !t.available || value < t.last {
		t.last = value
		t.available = true
		t.sampled = true
		return 0
	}
	delta := value - t.last
	t.last = value
	if active && delta > 0 {
		t.accumulated += delta
	}
	return delta
}

func (t validationIOTracker) estimate(total uint64) validationEstimate {
	if !t.available || total == 0 || t.accumulated == 0 {
		return validationEstimate{}
	}
	// Keep the estimate below 100%. Only Steam's terminal state is allowed to
	// promote the task to 100/completed.
	maxDone := total - 1
	if total >= 100 {
		maxDone = (total * 99) / 100
	}
	done := t.accumulated
	if done > maxDone {
		done = maxDone
	}
	progress := percent(done, total)
	if progress < 1 {
		progress = 1
	}
	return validationEstimate{Available: true, BytesDone: done, BytesTotal: total, Progress: progress}
}

func percent(done, total uint64) int {
	if total == 0 {
		return 0
	}
	p := int((done * 100) / total)
	if p < 0 {
		return 0
	}
	if p > 99 {
		return 99
	}
	return p
}
