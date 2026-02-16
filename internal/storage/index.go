package storage

import (
	"strconv"
	"strings"
)

// Filter defines optional criteria for querying tasks
type Filter struct {
	// Status filters tasks by status (e.g., "pending", "completed")
	Status string

	// Wave filters tasks by wave number
	Wave int

	// Complexity filters tasks by complexity level (e.g., "simple", "moderate", "complex")
	Complexity string

	// Tags filters tasks by tags (OR match - any tag matches)
	Tags []string

	// BlockedBy filters tasks by blocked status
	// If nil: no filtering by blocked status
	// If empty slice: only return unblocked tasks (tasks with empty BlockedBy)
	// If non-empty: return tasks blocked by specified task IDs
	BlockedBy []string
}

// matchesFilter checks if a task matches the filter criteria
func (t *Task) matchesFilter(f Filter) bool {
	// Filter by Status (AND)
	if f.Status != "" && t.Status != f.Status {
		return false
	}

	// Filter by Wave (AND)
	if f.Wave != 0 && t.Wave != f.Wave {
		return false
	}

	// Filter by Complexity (AND)
	if f.Complexity != "" && t.Complexity != f.Complexity {
		return false
	}

	// Filter by Tags (OR within tags)
	if len(f.Tags) > 0 {
		hasMatchingTag := false
		for _, filterTag := range f.Tags {
			for _, taskTag := range t.Tags {
				if taskTag == filterTag {
					hasMatchingTag = true
					break
				}
			}
			if hasMatchingTag {
				break
			}
		}
		if !hasMatchingTag {
			return false
		}
	}

	// Filter by BlockedBy (AND)
	// nil BlockedBy in filter means: don't filter by blocked status
	// empty slice in filter means: only return unblocked tasks
	// non-empty slice in filter means: return tasks blocked by those specific IDs
	if f.BlockedBy != nil {
		if len(f.BlockedBy) == 0 {
			// Filter for unblocked tasks only
			if len(t.BlockedBy) > 0 {
				return false
			}
		} else {
			// Filter for tasks blocked by specific IDs
			// Task must have at least one matching blocked ID
			hasMatchingBlocker := false
			for _, filterBlocker := range f.BlockedBy {
				for _, taskBlocker := range t.BlockedBy {
					if taskBlocker == filterBlocker {
						hasMatchingBlocker = true
						break
					}
				}
				if hasMatchingBlocker {
					break
				}
			}
			if !hasMatchingBlocker {
				return false
			}
		}
	}

	return true
}

// compareTaskIDs compares two task IDs for sorting
// Task IDs are hierarchical (e.g., "1.1", "1.2", "2.1")
// We split by "." and compare numerically
func compareTaskIDs(id1, id2 string) bool {
	parts1 := strings.Split(id1, ".")
	parts2 := strings.Split(id2, ".")

	// Compare each part numerically
	minLen := len(parts1)
	if len(parts2) < minLen {
		minLen = len(parts2)
	}

	for i := 0; i < minLen; i++ {
		// Parse as integers for proper numeric comparison
		n1, err1 := strconv.Atoi(parts1[i])
		if err1 != nil {
			n1 = 0
		}
		n2, err2 := strconv.Atoi(parts2[i])
		if err2 != nil {
			n2 = 0
		}

		if n1 < n2 {
			return true
		}
		if n1 > n2 {
			return false
		}
	}

	// If all parts are equal, shorter ID comes first
	return len(parts1) < len(parts2)
}
