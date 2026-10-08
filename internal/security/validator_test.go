package security

import (
	"testing"
)

func TestValidateSafeURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"Valid public https URL", "https://images.unsplash.com/photo-12345", false},
		{"Valid public http URL", "http://example.com/file.pdf", false},
		{"Loopback IP address", "http://127.0.0.1:8080/admin", true},
		{"Localhost domain", "http://localhost/test", true},
		{"Private 10.x.x.x network", "http://10.0.0.1/status", true},
		{"Private 192.168.x.x network", "http://192.168.1.1/router", true},
		{"Cloud metadata IP 169.254.169.254", "http://169.254.169.254/latest/meta-data/", true},
		{"Dangerous file protocol", "file:///etc/passwd", true},
		{"Dangerous javascript protocol", "javascript:alert(1)", true},
		{"Dangerous data protocol", "data:text/html,<script>alert(1)</script>", true},
		{"Empty URL", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSafeURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSafeURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeText(t *testing.T) {
	raw := `<script>alert('xss')</script>Hello & welcome!`
	sanitized := SanitizeText(raw)
	if sanitized == raw {
		t.Errorf("expected sanitized string to differ from raw script input")
	}
	expected := `&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;Hello &amp; welcome!`
	if sanitized != expected {
		t.Errorf("expected %q, got %q", expected, sanitized)
	}
}
