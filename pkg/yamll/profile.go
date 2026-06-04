package yamll

import (
	"fmt"
	"time"
)

// BuildProfile captures phase timings for build operations.
type BuildProfile struct {
	ImportResolution time.Duration
	RemoteFetch      time.Duration
	MergePhase       time.Duration
	Validation       time.Duration
	totalStart       time.Time
}

func (p *BuildProfile) Total() time.Duration {
	if p == nil || p.totalStart.IsZero() {
		return 0
	}

	return time.Since(p.totalStart)
}

func (p *BuildProfile) String() string {
	if p == nil {
		return ""
	}

	return fmt.Sprintf("Import resolution: %s\nRemote fetch: %s\nMerge phase: %s\nValidation: %s\nTotal: %s\n",
		prettyDuration(p.ImportResolution),
		prettyDuration(p.RemoteFetch),
		prettyDuration(p.MergePhase),
		prettyDuration(p.Validation),
		prettyDuration(p.Total()),
	)
}

const roundToMilliseconds = 10

func (p *BuildProfile) begin() {
	if p != nil && p.totalStart.IsZero() {
		p.totalStart = time.Now()
	}
}

func (p *BuildProfile) addRemoteFetch(duration time.Duration) {
	if p != nil {
		p.RemoteFetch += duration
	}
}

func (p *BuildProfile) addImportResolution(duration time.Duration) {
	if p != nil {
		p.ImportResolution += duration
	}
}

func (p *BuildProfile) addMerge(duration time.Duration) {
	if p != nil {
		p.MergePhase += duration
	}
}

func (p *BuildProfile) addValidation(duration time.Duration) {
	if p != nil {
		p.Validation += duration
	}
}

func prettyDuration(duration time.Duration) string {
	if duration < time.Millisecond {
		return duration.Round(time.Microsecond).String()
	}

	if duration < time.Second {
		return duration.Round(time.Millisecond).String()
	}

	return duration.Round(roundToMilliseconds * time.Millisecond).String()
}
