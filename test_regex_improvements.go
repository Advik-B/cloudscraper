package main

import (
	"fmt"
	"regexp"
	"strings"
)

func main() {
	fmt.Println("=== Testing Cloudflare Challenge Form Detection ===")
	fmt.Println("This test validates the regex improvements made to fix issue #2")
	fmt.Println()

	// Original regex that was causing the "could not find challenge form" error
	oldFormRegex := regexp.MustCompile(`<form class="challenge-form" id="challenge-form" action="(.+?)" method="POST">`)
	
	// New improved regex that should handle modern Cloudflare variations
	newFormRegex := regexp.MustCompile(`(?i)<form[^>]*?(?:class\s*=\s*"[^"]*challenge-form[^"]*"|id\s*=\s*"[^"]*challenge-form[^"]*")[^>]*?action\s*=\s*"([^"]+)"[^>]*?>`)
	
	// Test cases that represent various modern Cloudflare form patterns
	testCases := []struct {
		name string
		html string
		shouldWork bool
	}{
		{
			name: "Original format (should work with both)",
			html: `<form class="challenge-form" id="challenge-form" action="/cdn-cgi/l/chk_jschl" method="POST">`,
			shouldWork: true,
		},
		{
			name: "Reversed attributes (was failing)",
			html: `<form id="challenge-form" class="challenge-form" action="/cdn-cgi/l/chk_jschl" method="POST">`,
			shouldWork: true,
		},
		{
			name: "Extra CSS classes (was failing)",
			html: `<form class="challenge-form cf-form" id="challenge-form" action="/cdn-cgi/l/chk_jschl" method="POST">`,
			shouldWork: true,
		},
		{
			name: "Uppercase (was failing)",
			html: `<FORM CLASS="challenge-form" ID="challenge-form" ACTION="/cdn-cgi/l/chk_jschl" METHOD="POST">`,
			shouldWork: true,
		},
		{
			name: "Extra attributes (was failing)",
			html: `<form data-test="true" class="challenge-form" id="challenge-form" action="/cdn-cgi/l/chk_jschl" method="POST" novalidate>`,
			shouldWork: true,
		},
		{
			name: "Spaces around equals (was failing)",
			html: `<form class = "challenge-form" id = "challenge-form" action = "/cdn-cgi/l/chk_jschl" method = "POST">`,
			shouldWork: true,
		},
		{
			name: "Only ID, no class (was failing)",
			html: `<form id="challenge-form" action="/cdn-cgi/l/chk_jschl" method="POST">`,
			shouldWork: true,
		},
		{
			name: "Only class, no ID (was failing)",
			html: `<form class="challenge-form" action="/cdn-cgi/l/chk_jschl" method="POST">`,
			shouldWork: true,
		},
	}

	fmt.Printf("Testing %d different challenge form patterns...\n\n", len(testCases))

	oldFailures := 0
	newFailures := 0
	
	for i, testCase := range testCases {
		fmt.Printf("Test %d: %s\n", i+1, testCase.name)
		fmt.Printf("HTML: %s\n", testCase.html)
		
		oldMatch := oldFormRegex.FindStringSubmatch(testCase.html)
		newMatch := newFormRegex.FindStringSubmatch(testCase.html)
		
		oldWorks := len(oldMatch) > 1
		newWorks := len(newMatch) > 1
		
		fmt.Printf("Old regex: %s", formatResult(oldWorks))
		if oldWorks {
			fmt.Printf(" -> action: %s", oldMatch[1])
		} else if testCase.shouldWork {
			oldFailures++
		}
		fmt.Println()
		
		fmt.Printf("New regex: %s", formatResult(newWorks))
		if newWorks {
			fmt.Printf(" -> action: %s", newMatch[1])
		} else if testCase.shouldWork {
			newFailures++
		}
		fmt.Println()
		
		if testCase.shouldWork && !oldWorks && newWorks {
			fmt.Printf("✅ FIXED: This pattern now works with the new regex!\n")
		} else if testCase.shouldWork && oldWorks && newWorks {
			fmt.Printf("✅ MAINTAINED: This pattern continues to work\n")
		} else if testCase.shouldWork && !newWorks {
			fmt.Printf("❌ STILL BROKEN: This pattern should work but doesn't\n")
		}
		
		fmt.Println(strings.Repeat("-", 60))
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Old regex failures: %d/%d\n", oldFailures, len(testCases))
	fmt.Printf("New regex failures: %d/%d\n", newFailures, len(testCases))
	
	if newFailures == 0 {
		fmt.Printf("🎉 SUCCESS: All test cases pass with the new regex!\n")
		fmt.Printf("This should fix the 'v2: could not find challenge form' error.\n")
	} else {
		fmt.Printf("⚠️  WARNING: %d test cases still fail\n", newFailures)
	}

	// Test other improved regexes
	fmt.Printf("\n=== Testing Other Field Extraction Improvements ===\n")
	testOtherFields()
}

func formatResult(works bool) string {
	if works {
		return "✅ WORKS"
	}
	return "❌ FAILS"
}

func testOtherFields() {
	// Test improved jschl_vc and pass regexes
	oldJschlVc := regexp.MustCompile(`name="jschl_vc" value="(\w+)"`)
	newJschlVc := regexp.MustCompile(`(?i)name\s*=\s*"jschl_vc"\s+value\s*=\s*"([^"]+)"`)
	
	oldPass := regexp.MustCompile(`name="pass" value="(.+?)"`)
	newPass := regexp.MustCompile(`(?i)name\s*=\s*"pass"\s+value\s*=\s*"([^"]+)"`)
	
	testHtml := `<input type="hidden" NAME="jschl_vc" VALUE="abc123def" />
<input type="hidden" name = "pass" value = "xyz789" />`
	
	fmt.Printf("Testing field extraction with HTML:\n%s\n\n", testHtml)
	
	// Test jschl_vc
	oldVcMatch := oldJschlVc.FindStringSubmatch(testHtml)
	newVcMatch := newJschlVc.FindStringSubmatch(testHtml)
	
	fmt.Printf("jschl_vc field:\n")
	fmt.Printf("  Old regex: %s", formatResult(len(oldVcMatch) > 1))
	if len(oldVcMatch) > 1 {
		fmt.Printf(" -> %s", oldVcMatch[1])
	}
	fmt.Println()
	fmt.Printf("  New regex: %s", formatResult(len(newVcMatch) > 1))
	if len(newVcMatch) > 1 {
		fmt.Printf(" -> %s", newVcMatch[1])
	}
	fmt.Println()
	
	// Test pass
	oldPassMatch := oldPass.FindStringSubmatch(testHtml)
	newPassMatch := newPass.FindStringSubmatch(testHtml)
	
	fmt.Printf("pass field:\n")
	fmt.Printf("  Old regex: %s", formatResult(len(oldPassMatch) > 1))
	if len(oldPassMatch) > 1 {
		fmt.Printf(" -> %s", oldPassMatch[1])
	}
	fmt.Println()
	fmt.Printf("  New regex: %s", formatResult(len(newPassMatch) > 1))
	if len(newPassMatch) > 1 {
		fmt.Printf(" -> %s", newPassMatch[1])
	}
	fmt.Println()
}