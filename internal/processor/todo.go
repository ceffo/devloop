package processor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/yourusername/devloop/internal/agent"
	"github.com/yourusername/devloop/internal/config"
	"github.com/yourusername/devloop/internal/storage"
)

// TodoItem represents a single TODO item parsed from a markdown file.
type TodoItem struct {
	ID       string
	Category string
	Content  string
	Priority string
}

// ParseTodoFile parses a markdown TODO file and returns structured TodoItem entries.
// It extracts categories from headings, items from list entries, and priorities from markers.
//
// Priority markers:
//   - !! or !!! = high
//   - ! = medium
//   - - or no marker = low
//
// Completed items (- [x]) are skipped.
func ParseTodoFile(path string) ([]TodoItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open TODO file: %w", err)
	}
	defer file.Close()

	var items []TodoItem
	var currentCategory string
	itemCounter := 1

	// Regular expressions for parsing
	headingRegex := regexp.MustCompile(`^#+\s+(.+)$`)
	completedRegex := regexp.MustCompile(`^\s*[-*]\s+\[[xX]\]`)
	checkboxItemRegex := regexp.MustCompile(`^\s*[-*]\s+\[\s\]\s+(.+)$`)
	plainListItemRegex := regexp.MustCompile(`^\s*[-*]\s+(.+)$`)
	priorityRegex := regexp.MustCompile(`^(!!+|!)\s+`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Check for heading (category)
		if matches := headingRegex.FindStringSubmatch(trimmed); matches != nil {
			currentCategory = strings.TrimSpace(matches[1])
			continue
		}

		// Check for completed items - skip them
		if completedRegex.MatchString(line) {
			continue
		}

		// Check for unchecked checkbox items first
		var content string
		if matches := checkboxItemRegex.FindStringSubmatch(line); matches != nil {
			content = matches[1]
		} else if matches := plainListItemRegex.FindStringSubmatch(line); matches != nil {
			// Then check for plain list items
			content = matches[1]
		}

		if content != "" {
			// Extract priority
			priority := "low"
			if priorityMatches := priorityRegex.FindStringSubmatch(content); priorityMatches != nil {
				marker := priorityMatches[1]
				if strings.HasPrefix(marker, "!!") {
					priority = "high"
				} else if marker == "!" {
					priority = "medium"
				}
				// Remove priority marker from content
				content = priorityRegex.ReplaceAllString(content, "")
			}

			item := TodoItem{
				ID:       fmt.Sprintf("TODO-%d", itemCounter),
				Category: currentCategory,
				Content:  strings.TrimSpace(content),
				Priority: priority,
			}
			items = append(items, item)
			itemCounter++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading TODO file: %w", err)
	}

	return items, nil
}

// todoProcessingPromptTemplate is the template for converting TODO items into structured tasks
const todoProcessingPromptTemplate = `You are a task planning AI for the devloop system.

## Project Context
Name: {{.ProjectName}}
Tech Stack: {{.TechStack}}
Path: {{.ProjectPath}}

## Your Task
Analyze the following TODO items and convert them into structured, executable tasks.

## TODO Items
{{range .Todos}}
- [{{.ID}}] {{.Category}} | Priority: {{.Priority}}
  Content: {{.Content}}
{{end}}

## Instructions
1. Group related TODO items into logical tasks
2. Assign hierarchical task IDs starting from {{.NextID}}
3. For each task, determine:
   - Title (brief, imperative)
   - Description (detailed explanation)
   - Complexity: simple | moderate | complex
   - Acceptance criteria (3-5 testable outcomes)
   - Dependencies (blockedBy task IDs)
   - Tags (for categorization)

4. Output format: JSON array of task objects
   [
     {
       "id": "{{.NextID}}",
       "title": "...",
       "description": "...",
       "complexity": "simple|moderate|complex",
       "acceptance_criteria": ["...", "..."],
       "blocked_by": [],
       "tags": ["..."]
     }
   ]

## Guidelines
- simple: Single-file changes, < 50 lines
- moderate: Multi-file changes, 50-200 lines, state management
- complex: Algorithms, intricate workflows, architectural decisions
- Make tasks atomic and independently testable
- Clearly specify acceptance criteria
- Only add blockedBy if there's a true dependency

Output the JSON array only, no additional text.`

// todoPromptData holds data for rendering the TODO processing prompt
type todoPromptData struct {
	ProjectName string
	TechStack   string
	ProjectPath string
	Todos       []TodoItem
	NextID      string
}

// taskJSON represents the JSON structure returned by the AI agent
type taskJSON struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Complexity         string   `json:"complexity"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	BlockedBy          []string `json:"blocked_by"`
	Tags               []string `json:"tags"`
}

// renderTodoPrompt generates a prompt for processing TODO items into tasks
func renderTodoPrompt(project config.ProjectConfig, todos []TodoItem, nextID string) (string, error) {
	data := todoPromptData{
		ProjectName: project.Name,
		TechStack:   project.TechStack,
		ProjectPath: project.Path,
		Todos:       todos,
		NextID:      nextID,
	}

	tmpl, err := template.New("todo").Parse(todoProcessingPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse TODO prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render TODO prompt: %w", err)
	}

	return buf.String(), nil
}

// ProcessTodoItems converts TODO items into structured tasks using AI agent
// If review is true, tasks are displayed and user confirmation is required before returning
func ProcessTodoItems(cfg *config.Config, todos []TodoItem, review bool) ([]*storage.Task, error) {
	if len(todos) == 0 {
		return []*storage.Task{}, nil
	}

	// Create storage instance to query existing tasks
	store := storage.NewStorage(cfg)

	// Generate next available task ID
	nextID, err := generateNextTaskID(store)
	if err != nil {
		return nil, fmt.Errorf("failed to generate next task ID: %w", err)
	}

	// Render TodoProcessingPrompt with context
	prompt, err := renderTodoPrompt(cfg.Project, todos, nextID)
	if err != nil {
		return nil, fmt.Errorf("failed to render TODO prompt: %w", err)
	}

	// Create agent runner
	agentRunner, err := agent.NewAgentRunner(cfg.CLI.Tool)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent runner: %w", err)
	}

	// Execute Opus model (complex reasoning needed)
	logPath := filepath.Join(cfg.Project.Path, ".devloop", "logs", fmt.Sprintf("todo-processing-%d.log", time.Now().Unix()))
	model := cfg.CLI.Models["complex"] // Use Opus for complex reasoning
	result, err := agentRunner.Run(model, prompt, logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to run AI agent: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("AI agent execution failed: %v", result.Error)
	}

	// Parse JSON output (array of task objects)
	tasks, err := parseTasksFromJSON(result.Output, cfg, todos)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tasks from JSON: %w", err)
	}

	// If review mode, display tasks and prompt for confirmation
	if review {
		if !confirmTasks(tasks) {
			return nil, fmt.Errorf("task processing cancelled by user")
		}
	}

	return tasks, nil
}

// generateNextTaskID queries existing tasks and returns the next available ID
func generateNextTaskID(store *storage.Storage) (string, error) {
	tasks, err := store.LoadTasks()
	if err != nil {
		return "", fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(tasks) == 0 {
		return "1.1", nil
	}

	// Find the maximum task ID
	maxWave := 0
	maxTaskInWave := 0

	for _, task := range tasks {
		parts := strings.Split(task.ID, ".")
		if len(parts) != 2 {
			continue
		}

		wave, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		taskNum, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		if wave > maxWave {
			maxWave = wave
			maxTaskInWave = taskNum
		} else if wave == maxWave && taskNum > maxTaskInWave {
			maxTaskInWave = taskNum
		}
	}

	// Return next task ID in the same wave
	nextTaskNum := maxTaskInWave + 1
	return fmt.Sprintf("%d.%d", maxWave, nextTaskNum), nil
}

// parseTasksFromJSON extracts tasks from the AI agent's JSON output
func parseTasksFromJSON(output string, cfg *config.Config, todos []TodoItem) ([]*storage.Task, error) {
	// Extract JSON array from output (AI might include additional text)
	// Look for the JSON array pattern [...]
	start := strings.Index(output, "[")
	end := strings.LastIndex(output, "]")

	if start == -1 || end == -1 || start > end {
		return nil, fmt.Errorf("no valid JSON array found in agent output")
	}

	jsonStr := output[start : end+1]

	// Parse JSON array
	var taskJSONs []taskJSON
	if err := json.Unmarshal([]byte(jsonStr), &taskJSONs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert to Task structs
	var tasks []*storage.Task
	now := time.Now()

	for _, tj := range taskJSONs {
		// Determine wave from task ID
		wave := extractWaveFromID(tj.ID)

		// Assign model based on complexity
		model := cfg.CLI.Models[tj.Complexity]
		if model == "" {
			// Default to moderate if complexity not recognized
			model = cfg.CLI.Models["moderate"]
		}

		// Find source TODO item (if any)
		sourceTodoItem := ""
		if len(todos) > 0 {
			// Use first TODO item as source (simplified approach)
			sourceTodoItem = todos[0].ID
		}

		task := &storage.Task{
			ID:                 tj.ID,
			Title:              tj.Title,
			Wave:               wave,
			Status:             "pending",
			Complexity:         tj.Complexity,
			Model:              model,
			Description:        tj.Description,
			AcceptanceCriteria: tj.AcceptanceCriteria,
			BlockedBy:          tj.BlockedBy,
			Tags:               tj.Tags,
			Metadata: storage.TaskMetadata{
				CreatedAt:      now,
				UpdatedAt:      now,
				SourceType:     "todo",
				SourceTodoItem: sourceTodoItem,
				MaxAttempts:    cfg.Execution.MaxAttempts,
			},
			Execution: storage.TaskExecution{
				Attempts:      []storage.Attempt{},
				TotalDuration: 0,
			},
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// extractWaveFromID extracts the wave number from a task ID (e.g., "1.1" -> 1)
func extractWaveFromID(id string) int {
	parts := strings.Split(id, ".")
	if len(parts) != 2 {
		return 1
	}

	wave, err := strconv.Atoi(parts[0])
	if err != nil {
		return 1
	}

	return wave
}

// confirmTasks displays tasks and prompts user for confirmation
func confirmTasks(tasks []*storage.Task) bool {
	fmt.Println("\n=== Generated Tasks ===")
	for _, task := range tasks {
		fmt.Printf("\n[%s] %s\n", task.ID, task.Title)
		fmt.Printf("  Complexity: %s\n", task.Complexity)
		fmt.Printf("  Model: %s\n", task.Model)
		fmt.Printf("  Description: %s\n", task.Description)
		if len(task.AcceptanceCriteria) > 0 {
			fmt.Println("  Acceptance Criteria:")
			for _, criterion := range task.AcceptanceCriteria {
				fmt.Printf("    - %s\n", criterion)
			}
		}
		if len(task.BlockedBy) > 0 {
			fmt.Printf("  Blocked by: %v\n", task.BlockedBy)
		}
		if len(task.Tags) > 0 {
			fmt.Printf("  Tags: %v\n", task.Tags)
		}
	}

	fmt.Printf("\nGenerated %d task(s). Save to storage? [y/N]: ", len(tasks))

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
