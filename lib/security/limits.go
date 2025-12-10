package security

import "fmt"

const (
// MaxExternalScriptSize is the maximum script size for external JS engines to prevent DoS attacks (5MB)
MaxExternalScriptSize = 5 * 1024 * 1024
// MaxGojaScriptSize is the maximum script size for Goja engine to prevent DoS attacks (5MB)
MaxGojaScriptSize = 5 * 1024 * 1024
// MaxChallengeScriptSize is the maximum size for challenge scripts to prevent DoS attacks (1MB)
MaxChallengeScriptSize = 1024 * 1024
)

// ValidateScriptSize checks if the script size is within the allowed limit.
// Returns an error if the script exceeds the maximum size.
func ValidateScriptSize(script string, maxSize int) error {
if len(script) > maxSize {
return fmt.Errorf("script size exceeds maximum allowed size (%d bytes)", maxSize)
}
return nil
}

// ValidateTotalScriptSize checks if the total size of multiple script matches is within the allowed limit.
// Returns an error if the total size exceeds the maximum.
func ValidateTotalScriptSize(scriptMatches [][]string, maxSize int) error {
totalSize := 0
for _, match := range scriptMatches {
if len(match) > 1 {
totalSize += len(match[1])
}
}
if totalSize > maxSize {
return fmt.Errorf("total script size exceeds maximum allowed size (%d bytes)", maxSize)
}
return nil
}
