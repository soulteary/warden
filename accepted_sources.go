package main

import (
	"github.com/soulteary/warden/internal/loader"
	"sync/atomic"
	"time"
)

var acceptedSourceGeneration atomic.Uint64

// acceptedSourceEvidence belongs to one accepted in-memory publication. Generation
// is process-local, not an ordered upstream version or persisted Redis revision.
// Acceptance does not assert that every individual input passed cache validation.
type acceptedSourceEvidence struct {
	AcceptedAt       time.Time
	SourceSetID      string
	InputVersion     string
	EffectiveVersion string
	Manifests        []loader.SourceManifest
	Attempts         []loader.SourceAttempt
	Generation       uint64
}

func cloneAcceptedSources(input *acceptedSourceEvidence) *acceptedSourceEvidence {
	if input == nil {
		return nil
	}
	out := *input
	out.Manifests = loader.CloneManifests(input.Manifests)
	out.Attempts = append([]loader.SourceAttempt(nil), input.Attempts...)
	return &out
}
