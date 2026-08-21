package identity

import "strings"

// MaskPhone masks a phone number, keeping only the last two digits.
// Empty input returns an empty string. Example: "13800138000" -> "*********00".
func MaskPhone(phone string) string {
	p := strings.TrimSpace(phone)
	if p == "" {
		return ""
	}
	if len(p) <= 2 {
		return strings.Repeat("*", len(p))
	}
	return strings.Repeat("*", len(p)-2) + p[len(p)-2:]
}

// MaskMail masks an email, keeping the first character of the local part and the
// full domain. Example: "alice@example.com" -> "a****@example.com".
func MaskMail(mail string) string {
	m := strings.ToLower(strings.TrimSpace(mail))
	if m == "" {
		return ""
	}
	at := strings.LastIndex(m, "@")
	if at <= 0 {
		// No usable local part; mask everything but keep length signal small.
		if len(m) <= 1 {
			return "*"
		}
		return m[:1] + strings.Repeat("*", len(m)-1)
	}
	local := m[:at]
	domain := m[at:]
	if len(local) == 1 {
		return local + "*" + domain
	}
	return local[:1] + strings.Repeat("*", len(local)-1) + domain
}

// MaskUserID masks a derived/opaque user_id, keeping a short non-identifying prefix.
func MaskUserID(id string) string {
	s := strings.TrimSpace(id)
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}

// maskIdentifier masks whichever identifier is present (phone preferred, else mail).
func maskIdentifier(phone, mail string) string {
	if strings.TrimSpace(phone) != "" {
		return MaskPhone(phone)
	}
	return MaskMail(mail)
}
