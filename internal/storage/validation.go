package storage

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DetectIDFormat returns "hierarchical" or "jira" based on ID format
func DetectIDFormat(id string) string {
	if IsValidJiraID(id) {
		return "jira"
	}
	if IsValidHierarchicalID(id) {
		return "hierarchical"
	}
	return "unknown"
}

// IsValidHierarchicalID checks if ID matches hierarchical format: N.N (e.g., "1.1", "2.5")
func IsValidHierarchicalID(id string) bool {
	parts := strings.Split(id, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

// IsValidJiraID checks if ID matches JIRA format: PREFIX-N (e.g., "DEV-123")
func IsValidJiraID(id string) bool {
	pattern := `^[A-Z]+[A-Z0-9]*-\d+$`
	matched, _ := regexp.MatchString(pattern, id)
	return matched
}

// CompareTaskIDsUniversal handles both hierarchical and JIRA IDs for sorting
// Returns true if id1 should come before id2
func CompareTaskIDsUniversal(id1, id2 string) bool {
	format1 := DetectIDFormat(id1)
	format2 := DetectIDFormat(id2)

	// Both hierarchical: use hierarchical comparison
	if format1 == "hierarchical" && format2 == "hierarchical" {
		return compareTaskIDsHierarchical(id1, id2)
	}

	// Both JIRA: compare by number
	if format1 == "jira" && format2 == "jira" {
		return compareJiraIDs(id1, id2)
	}

	// Mixed: hierarchical comes before JIRA (chronological order)
	return format1 == "hierarchical"
}

// compareTaskIDsHierarchical compares two hierarchical IDs (e.g., "1.1" vs "2.3")
// Returns true if id1 should come before id2
func compareTaskIDsHierarchical(id1, id2 string) bool {
	parts1 := strings.Split(id1, ".")
	parts2 := strings.Split(id2, ".")

	// Compare level by level
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		num1, _ := strconv.Atoi(parts1[i])
		num2, _ := strconv.Atoi(parts2[i])

		if num1 != num2 {
			return num1 < num2
		}
	}

	// If all compared parts are equal, shorter ID comes first
	return len(parts1) < len(parts2)
}

// compareJiraIDs compares two JIRA IDs (e.g., "DEV-1" vs "DEV-15")
// Returns true if id1 should come before id2
func compareJiraIDs(id1, id2 string) bool {
	_, num1, _ := parseJiraID(id1)
	_, num2, _ := parseJiraID(id2)
	return num1 < num2
}

// parseJiraID parses a JIRA ID into prefix and number
func parseJiraID(id string) (prefix string, num int, err error) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid JIRA ID format: %s", id)
	}
	num, err = strconv.Atoi(parts[1])
	return parts[0], num, err
}
