package word

import (
	"math/rand/v2"
	"testing"
)

func TestPalindrome(t *testing.T) {
	var tests = []struct {
		input string
		want  bool
	}{
		{"kajak", true},
		{"test", false},
		{"owocowo", true},
		{"A man, a plan, a canal: Panama", true},
		{"Evil I did dwell; lewd did I live.", true},
		{"Able was I ere I saw Elba", true},
		{"été", true},
		{"Et se resservir, ivresse reste.", true},
		{"palindrom", false}, // to nie jest palindrom
		{"żartem,", false},   // to jest półpalindrom

	}

	for _, test := range tests {
		if test.want != IsPalindrome(test.input) {
			t.Errorf("IsPalindrome(%q) = %v; want %v", test.input, !test.want, test.want)
		}
	}

}

func randomPalindrome() string {
	n := rand.IntN(25)
	runes := make([]rune, n)
	for i := 0; i < n; i++ {
		r := rune(rand.IntN(0x1000))
		runes[i] = r
		// runes[n-1-i] = r
	}
	return string(runes)
}

func TestRandomPalindromes(t *testing.T) {

	for range 100 {
		p := randomPalindrome()
		if !IsPalindrome(p) {
			t.Errorf("IsPalindrome(%q) = false", p)
		}
	}

}

// func TestFrenchPalindrome(t *testing.T) {
// 	if !IsPalindrome("été") {
// 		t.Error(`IsPalindrome("été") = false`)
// 	}
// }
// func TestCanalPalindrome(t *testing.T) {
// 	input := "A man, a plan, a canal: Panama"
// 	if !IsPalindrome(input) {
// 		t.Errorf(`IsPalindrome(%q) = false`, input)
// 	}
// }

// func TestNotPalindrome(t *testing.T) {
// 	if IsPalindrome("hello") {
// 		t.Error(`IsPalindrome("hello") = true, want false`)
// 	}
// }
