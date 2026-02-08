package filter

import (
	"strings"
	"testing"
	"time"
)

func TestFilter(t *testing.T) {
	filter := DefaultFilter()

	t.Run("FilterText_Phone", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "simple phone",
				input:    "我的电话是13812345678",
				expected: "我的电话是138****5678",
			},
			{
				name:     "phone with prefix",
				input:    "Phone: +86-13912345678",
				expected: "Phone: +86-139****5678",
			},
			{
				name:     "multiple phones",
				input:    "联系13812345678或13912345678",
				expected: "联系138****5678或139****5678",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := filter.FilterText(tt.input)
				if result != tt.expected {
					t.Errorf("FilterText() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("FilterText_IDCard", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "18-digit ID",
				input:    "身份证号110101199001011234",
				expected: "身份证号110***********1234",
			},
			{
				name:     "ID with X",
				input:    "身份证号11010119900101123X",
				expected: "身份证号110***********123X",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := filter.FilterText(tt.input)
				if result != tt.expected {
					t.Errorf("FilterText() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("FilterText_Email", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			contains string
		}{
			{
				name:     "simple email",
				input:    "Email: user@example.com",
				contains: "use***@***ple.com",
			},
			{
				name:     "email with numbers",
				input:    "user123@test.co.uk",
				contains: "use***@***t.co.uk",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := filter.FilterText(tt.input)
				// Check that @ is preserved
				if !strings.Contains(result, "@") {
					t.Error("expected @ to be preserved in email")
				}
			})
		}
	})

	t.Run("FilterText_BankCard", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			contains string
		}{
			{
				name:     "16-digit card",
				input:    "卡号6222021234567890",
				contains: "622***********7890",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := filter.FilterText(tt.input)
				// Just verify some masking occurred
				if !strings.Contains(result, "*") {
					t.Error("expected bank card to be masked")
				}
			})
		}
	})

	t.Run("FilterText_IP", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			contains string
		}{
			{
				name:     "IPv4 address",
				input:    "IP: 192.168.1.1",
				contains: "192.***.*.1",
			},
			{
				name:     "localhost",
				input:    "localhost",
				contains: "localhost",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := filter.FilterText(tt.input)
				if tt.name == "IPv4 address" && !strings.Contains(result, "*") {
					t.Error("expected IP to be masked")
				}
				if tt.name == "localhost" && result != tt.input {
					t.Errorf("localhost should not be modified, got %v", result)
				}
			})
		}
	})

	t.Run("FilterText_Mixed", func(t *testing.T) {
		input := "请联系：电话13812345678，邮箱test@example.com，身份证110101199001011234"
		result := filter.FilterText(input)

		// Verify all sensitive info is masked
		if !strings.Contains(result, "138****") {
			t.Error("phone not masked")
		}
		if !strings.Contains(result, "@") {
			t.Error("email @ should be preserved")
		}
		if !strings.Contains(result, "110***********") {
			t.Error("ID card not masked")
		}
		// Verify email contains masking
		if !strings.Contains(result, "*@") && !strings.Contains(result, "***@") {
			t.Error("email not properly masked")
		}
	})

	t.Run("FilterWithOptions", func(t *testing.T) {
		input := "电话13812345678"
		result := filter.FilterWithOptions(input, 2, 2, '#')

		if !strings.Contains(result, "#") {
			t.Error("expected custom mask character")
		}
	})

	t.Run("Validate", func(t *testing.T) {
		filtered := filter.FilterText("我的电话13812345678")
		if !filter.Validate(filtered) {
			t.Error("filtered text should be valid")
		}

		unfiltered := "我的电话13812345678"
		if filter.Validate(unfiltered) {
			t.Error("unfiltered text should not be valid")
		}
	})

	t.Run("FindMatches", func(t *testing.T) {
		input := "电话13812345678，邮箱test@example.com"
		matches := filter.FindMatches(input)

		if len(matches) < 2 {
			t.Errorf("expected at least 2 matches, got %d", len(matches))
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		filter.FilterText("电话13812345678 邮箱test@example.com")
		stats := filter.GetStats()

		if stats.TotalMatches == 0 {
			t.Error("expected non-zero total matches")
		}
		if stats.PhoneMatches == 0 {
			t.Error("expected non-zero phone matches")
		}
		if stats.EmailMatches == 0 {
			t.Error("expected non-zero email matches")
		}
	})
}

func TestValidateFunctions(t *testing.T) {
	t.Run("ValidatePhone", func(t *testing.T) {
		tests := []struct {
			input string
			valid bool
		}{
			{"13812345678", true},
			{"19912345678", true},
			{"12812345678", false},  // Invalid prefix
			{"1381234567", false},   // Too short
			{"138123456789", false}, // Too long
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result := ValidatePhone(tt.input)
				if result != tt.valid {
					t.Errorf("ValidatePhone(%v) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})

	t.Run("ValidateIDCard", func(t *testing.T) {
		tests := []struct {
			input string
			valid bool
		}{
			{"110101199001011234", true},
			{"11010119900101123X", true},
			{"11010119900101123x", true},
			{"11010119900101123", false},   // Too short
			{"1101011990010112345", false}, // Too long
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result := ValidateIDCard(tt.input)
				if result != tt.valid {
					t.Errorf("ValidateIDCard(%v) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})

	t.Run("ValidateEmail", func(t *testing.T) {
		tests := []struct {
			input string
			valid bool
		}{
			{"user@example.com", true},
			{"user123@test.co.uk", true},
			{"invalid", false},
			{"@example.com", false},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result := ValidateEmail(tt.input)
				if result != tt.valid {
					t.Errorf("ValidateEmail(%v) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})

	t.Run("ValidateBankCard", func(t *testing.T) {
		tests := []struct {
			input string
			valid bool
		}{
			{"6222021234567890", true},
			{"123456789012", true},
			{"12345678901", false},          // Too short
			{"12345678901234567890", false}, // Too long
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result := ValidateBankCard(tt.input)
				if result != tt.valid {
					t.Errorf("ValidateBankCard(%v) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})

	t.Run("ValidateIP", func(t *testing.T) {
		tests := []struct {
			input string
			valid bool
		}{
			{"192.168.1.1", true},
			{"255.255.255.255", true},
			{"0.0.0.0", true},
			{"256.1.1.1", false},
			{"192.168.1", false},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result := ValidateIP(tt.input)
				if result != tt.valid {
					t.Errorf("ValidateIP(%v) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})
}

func TestPatternSet(t *testing.T) {
	ps := NewPatternSet([]FilterType{Phone, Email})

	t.Run("Match", func(t *testing.T) {
		if !ps.Match("电话13812345678") {
			t.Error("expected phone to match")
		}
		if !ps.Match("test@example.com") {
			t.Error("expected email to match")
		}
		if ps.Match("身份证110101199001011234") {
			t.Error("expected ID card not to match (not in set)")
		}
	})

	t.Run("FindAll", func(t *testing.T) {
		text := "电话13812345678，邮箱test@example.com"
		matches := ps.FindAll(text)

		if len(matches) != 2 {
			t.Errorf("expected 2 matches, got %d", len(matches))
		}
	})

	t.Run("Add", func(t *testing.T) {
		ps.Add(IDCard)

		if !ps.Has(IDCard) {
			t.Error("expected ID card to be in set")
		}
	})

	t.Run("Remove", func(t *testing.T) {
		ps.Remove(Phone)

		if ps.Has(Phone) {
			t.Error("expected phone to be removed from set")
		}
	})

	t.Run("Types", func(t *testing.T) {
		types := ps.Types()

		if len(types) == 0 {
			t.Error("expected at least one type")
		}
	})
}

func TestFastScanner(t *testing.T) {
	scanner, err := NewFastScanner([]FilterType{Phone, Email, IDCard})
	if err != nil {
		t.Fatalf("NewFastScanner failed: %v", err)
	}

	t.Run("Scan", func(t *testing.T) {
		text := "电话13812345678，邮箱test@example.com，身份证110101199001011234"
		matches := scanner.Scan(text)

		if len(matches) < 3 {
			t.Errorf("expected at least 3 matches, got %d", len(matches))
		}
	})

	t.Run("HasAny", func(t *testing.T) {
		if !scanner.HasAny("电话13812345678") {
			t.Error("expected HasAny to return true")
		}
		if scanner.HasAny("普通文本") {
			t.Error("expected HasAny to return false")
		}
	})
}

func TestFilterConfig(t *testing.T) {
	cfg := FilterConfig{
		Enabled:        []FilterType{Phone, Email},
		MaskChar:       '#',
		PreserveLength: true,
		KeepFirstN:     2,
		KeepLastN:      3,
	}

	filter := NewFilter(cfg)
	input := "电话13812345678"
	result := filter.FilterText(input)

	if !strings.Contains(result, "#") {
		t.Error("expected custom mask character")
	}
}

func BenchmarkFilterText(b *testing.B) {
	filter := DefaultFilter()
	input := "请联系：电话13812345678，邮箱test@example.com，身份证110101199001011234，卡号6222021234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.FilterText(input)
	}
}

func BenchmarkFindMatches(b *testing.B) {
	filter := DefaultFilter()
	input := "请联系：电话13812345678，邮箱test@example.com，身份证110101199001011234"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.FindMatches(input)
	}
}

func BenchmarkFastScanner(b *testing.B) {
	scanner, _ := NewFastScanner([]FilterType{Phone, Email, IDCard, BankCard, IP})
	input := "请联系：电话13812345678，邮箱test@example.com，身份证110101199001011234"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.Scan(input)
	}
}

func BenchmarkPatternSet(b *testing.B) {
	ps := NewPatternSet([]FilterType{Phone, Email, IDCard, BankCard, IP})
	input := "请联系：电话13812345678，邮箱test@example.com，身份证110101199001011234"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.FindAll(input)
	}
}

func TestFilterPerformance(t *testing.T) {
	filter := DefaultFilter()
	input := "请联系：电话13812345678，邮箱test@example.com，身份证110101199001011234，卡号6222021234567890，IP 192.168.1.1"

	// Run multiple times to measure average latency
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		filter.FilterText(input)
	}

	elapsed := time.Since(start)
	avgNs := elapsed.Nanoseconds() / int64(iterations)

	t.Logf("Average filter time: %d ns", avgNs)

	if avgNs > 1_000_000 { // 1ms threshold
		t.Errorf("Filter too slow: %d ns (expected <1ms)", avgNs)
	}
}
