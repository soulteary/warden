package main

import "github.com/soulteary/warden/internal/loader"

// Source-attempt diagnostics describe loader selection, not cache acceptance or
// source freshness. No source URL, file path, credentials or user data is logged.
func (app *App) recordSourceAttempts(result *loader.LoadResult) {
	if app.log == nil || result == nil {
		return
	}
	for i := range result.Attempts {
		attempt := &result.Attempts[i]
		app.log.Debug().
			Int("attempt", attempt.Attempt).
			Str("source_id", attempt.SourceID).
			Str("source_kind", string(attempt.Kind)).
			Str("transport", attempt.Transport).
			Str("outcome", attempt.Outcome).
			Str("reason", attempt.Reason).
			Time("started_at", attempt.StartedAt).
			Time("finished_at", attempt.FinishedAt).
			Int("decoded_rows", attempt.Rows).
			Bool("selected_input", attempt.SelectedInput).
			Int("winning_rows", attempt.WinningRows).
			Bool("load_succeeded", result.Err == nil).
			Msg("source attempt completed")
	}
}
