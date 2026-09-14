package utils_test

import (
	"fmt"
	"testing"

	"github.com/ivaquero/gotmail/utils"
)

func TestGenerateRandomString(t *testing.T) {
	fmt.Println("=== Test random string generation ===")

	// Generate multiple random strings to verify functionality
	for i := 0; i < 5; i++ {
		randomStr, err := utils.GenerateRandomString(10)
		if err != nil {
			t.Fatalf("GenerateRandomString(10) returned an error: %v", err)
		}
		fmt.Printf("Random string %d: %s\n", i+1, randomStr)

		// Verify string length
		if len(randomStr) != 10 {
			t.Errorf("Generated string length incorrect: expected 10, actual %d", len(randomStr))
		}

		// Verify string only contains allowed characters
		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
		for _, char := range randomStr {
			found := false
			for _, allowed := range charset {
				if char == allowed {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Generated string contains invalid character: %c", char)
			}
		}
	}

	// A collapsed generator would still pass the length and charset checks
	// above -- the old deterministic fallback returned "abcdefghij" every
	// single time -- so require two draws to differ as well.
	first, err := utils.GenerateRandomString(10)
	if err != nil {
		t.Fatalf("GenerateRandomString(10) returned an error: %v", err)
	}
	second, err := utils.GenerateRandomString(10)
	if err != nil {
		t.Fatalf("GenerateRandomString(10) returned an error: %v", err)
	}
	if first == second {
		t.Errorf("Two consecutive draws are identical (%q); randomness has collapsed", first)
	}

	fmt.Println("\n=== Test completed ===")
}

// TestGenerateRandomStringRejectsNonPositiveLength covers B-2: this function
// mints account passwords, so a bad length must fail loudly instead of
// yielding an empty password (length 0) or panicking inside make() (negative).
func TestGenerateRandomStringRejectsNonPositiveLength(t *testing.T) {
	for _, length := range []int{0, -1} {
		if _, err := utils.GenerateRandomString(length); err == nil {
			t.Errorf("Expected an error for length %d, got nil", length)
		}
	}
}
