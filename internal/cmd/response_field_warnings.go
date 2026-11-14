package cmd

// ResponseFieldWarnings reports a response field allowlist that leaves out
// "status". The user responses then carry no status, and a caller that decides
// on it -- the SDK's CheckUserInList does -- reads every user as not active.
// The configuration is accepted: a deployment that only lists users, or that
// decides through /v1/lookup, which always returns the status, does not need
// the field.
func ResponseFieldWarnings(cfg *Config) []string {
	if len(cfg.ResponseFields) == 0 {
		return nil
	}
	for _, field := range cfg.ResponseFields {
		if field == "status" {
			return nil
		}
	}
	return []string{"RESPONSE_FIELDS does not include status: GET /user and the list responses carry no status, so a client that checks it treats every user as not active (the SDK's CheckUserInList returns false and CheckUserInListWithError returns an error); add status to RESPONSE_FIELDS, or decide through /v1/lookup, which always returns it"}
}
