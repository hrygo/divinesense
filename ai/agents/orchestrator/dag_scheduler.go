package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Task execution timeout - configurable via environment variable or config
const defaultTaskTimeout = 120 * time.Second

// DAGScheduler handles dependency-based task execution.
// It implements Kahn's Algorithm for dynamic task dispatching.
type DAGScheduler struct {
	// Task management
	tasks       map[string]*Task
	graph       map[string][]string // upstream -> downstreams
	inDegree    map[string]int      // task -> remaining dependencies count
	taskIndices map[string]int      // taskID -> original index

	// Execution state
	readyQueue chan string // Tasks ready to run (inDegree = 0)

	// Synchronization
	mu            sync.Mutex
	activeWorkers int

	// Component access
	executor   *Executor
	injector   *ContextInjector
	dispatcher *EventDispatcher

	// Observability
	traceID string

	// Timeout
	taskTimeout time.Duration
}

func NewDAGScheduler(executor *Executor, tasks []*Task, traceID string, dispatcher *EventDispatcher) (*DAGScheduler, error) {
	s := &DAGScheduler{
		tasks:       make(map[string]*Task),
		graph:       make(map[string][]string),
		inDegree:    make(map[string]int),
		taskIndices: make(map[string]int),
		readyQueue:  make(chan string, len(tasks)), // Buffer large enough for all tasks
		executor:    executor,
		injector:    NewContextInjector(),
		traceID:     traceID,
		dispatcher:  dispatcher,
		taskTimeout: defaultTaskTimeout,
	}

	// Initialize state
	for i, task := range tasks {
		s.tasks[task.ID] = task
		s.taskIndices[task.ID] = i
		s.inDegree[task.ID] = 0 // Init to 0
	}

	// Build dependency graph
	for _, task := range tasks {
		for _, depID := range task.Dependencies {
			if _, exists := s.tasks[depID]; !exists {
				return nil, fmt.Errorf("task %s depends on unknown task %s", task.ID, depID)
			}

			s.graph[depID] = append(s.graph[depID], task.ID)
			s.inDegree[task.ID]++
		}
	}

	// Seed ready queue
	for _, task := range tasks {
		if s.inDegree[task.ID] == 0 {
			s.readyQueue <- task.ID
		}
	}

	// Detect cycles before returning
	if err := s.detectCycle(); err != nil {
		return nil, fmt.Errorf("cycle detected in task dependencies: %w", err)
	}

	return s, nil
}

// detectCycle uses DFS to detect if there is a cycle in the DAG.
// Returns an error if a cycle is found, with the first node involved in the cycle.
func (s *DAGScheduler) detectCycle() error {
	// Use three-color DFS:
	// 0 = unvisited, 1 = visiting (in current path), 2 = visited
	color := make(map[string]int)

	var dfs func(node string) error
	dfs = func(node string) error {
		color[node] = 1 // Mark as visiting

		// Visit all downstream nodes
		for _, downstream := range s.graph[node] {
			if color[downstream] == 1 {
				// Found a cycle
				return fmt.Errorf("cycle at task %s -> %s", node, downstream)
			}
			if color[downstream] == 0 {
				if err := dfs(downstream); err != nil {
					return err
				}
			}
		}

		color[node] = 2 // Mark as visited
		return nil
	}

	// Start DFS from all nodes
	for node := range s.tasks {
		if color[node] == 0 {
			if err := dfs(node); err != nil {
				return err
			}
		}
	}

	return nil
}

// Run starts the scheduling loop.
func (s *DAGScheduler) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	// For MVP, we use a dispatcher loop that feeds a semaphore-controlled worker pool.
	sem := make(chan struct{}, s.executor.config.MaxParallelTasks)

	// Log DAG schedule start
	slog.Info("executor: dag schedule",
		"trace_id", s.traceID,
		"ready_tasks", len(s.tasks),
		"active_tasks", 0,
	)

	// Orchestration loop
	for {
		select {
		case <-ctx.Done():
			wg.Wait() // Wait for all running goroutines to complete
			return ctx.Err()
		case err := <-errChan:
			wg.Wait() // Wait for all running goroutines to complete
			return err
		case taskID := <-s.readyQueue:
			s.mu.Lock()
			s.activeWorkers++
			s.mu.Unlock()

			wg.Add(1)

			go func(tid string) {
				defer wg.Done()
				sem <- struct{}{}        // Acquire token (blocks if full)
				defer func() { <-sem }() // Release token

				// Wrap execution in a function to handle panics and state updates safely
				func() {
					defer func() {
						// Handle panic
						if r := recover(); r != nil {
							slog.Error("executor: panic in task execution",
								"trace_id", s.traceID,
								"task_id", tid,
								"panic", r)

							// Mark task as failed and cascade
							// Note: s.mu protects DAG state (tasks map, inDegree, etc)
							// but Task internal state is protected by its own mutex.
							// However, s.cascadeSkip reads/writes task status too.
							// We need to be careful about lock ordering if we hold both.
							// Here we hold s.mu.

							task := s.tasks[tid]
							// Use safe accessor - though we are holding s.mu, other goroutines (Executor)
							// might be modifying Task without s.mu.
							if !task.GetStatus().IsTerminal() {
								task.SetError(fmt.Sprintf("Panic: %v", r))
								s.cascadeSkip(tid)
							}
						}

						// Always decrement active workers
						s.mu.Lock()
						s.activeWorkers--

						// Log DAG schedule state after task completion
						readyCount := len(s.readyQueue)
						slog.Info("executor: dag schedule",
							"trace_id", s.traceID,
							"ready_tasks", readyCount,
							"active_tasks", s.activeWorkers,
						)
						s.mu.Unlock()
					}()

					// Execute task
					err := s.executeTask(ctx, tid)
					if err != nil {
						// Log error but don't stop the world unless strict mode
						// This allows partial failure in DAG execution
						slog.Warn("DAG task execution failed",
							"trace_id", s.traceID,
							"task_id", tid,
							"error", err.Error())
					}
				}()
			}(taskID)

		default:
			// Check for completion or deadlock
			s.mu.Lock()
			active := s.activeWorkers
			completed := 0
			for _, t := range s.tasks {
				// Use safe accessor - accessing Task status is safe, but iterating map needs s.mu?
				// s.tasks map structure is immutable after Init? Yes.
				// But we are holding s.mu here to protect activeWorkers and to consistent view?
				// Actually s.mu protects the *Scheduler* state.
				if t.GetStatus().IsTerminal() {
					completed++
				}
			}

			if completed == len(s.tasks) {
				s.mu.Unlock()
				wg.Wait() // Ensure all workers fully exit
				return nil
			}

			if active == 0 {
				s.mu.Unlock()
				// No active workers and ready queue is empty (implied by default case)
				// This means we are stuck -> Cycle detected
				wg.Wait() // Wait for all running goroutines to complete
				return fmt.Errorf("cycle detected or deadlock: %d/%d tasks completed", completed, len(s.tasks))
			}
			s.mu.Unlock()

			// Avoid hot loop - use 50ms to reduce CPU usage while remaining responsive
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// executeTask handles the actual execution logic
func (s *DAGScheduler) executeTask(ctx context.Context, taskID string) error {
	task := s.tasks[taskID]

	// Check if task is already terminal (e.g., skipped or previously completed)
	// This avoids unnecessary processing and race conditions
	if task.GetStatus().IsTerminal() {
		return nil
	}

	// 1. Context Injection
	resolvedInput, err := s.injector.ResolveInput(task.Input, s.tasks)
	if err != nil {
		// Injection failed (e.g. ref not found)
		s.mu.Lock() // Consistently lock s.mu for coordination
		task.SetError(fmt.Sprintf("Context Injection Error: %v", err))
		s.cascadeSkip(taskID)
		s.mu.Unlock()
		return err
	}

	// Task.Input is not protected by mutex currently as it's modified only before execution
	// But strictly speaking we should probably protect it or treat it as immutable once created,
	// except here we are resolving it. Since only one worker processes this task, it's fine.
	task.Input = resolvedInput

	// 2. Execute with timeout
	// Use the event dispatcher for safe event emission
	taskIndex := s.taskIndices[taskID]

	// Create a context with timeout for task execution
	taskCtx, cancel := context.WithTimeout(ctx, s.taskTimeout)
	defer cancel()

	err = s.executor.executeTask(taskCtx, task, taskIndex, s.dispatcher, s.traceID)

	// Check if the task timed out
	if err == nil && taskCtx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("task execution timed out after %v", s.taskTimeout)
		task.SetError(err.Error())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		// Task Failed Logic (Executor already set Status/Error)
		// Cascade Skip
		s.cascadeSkip(taskID)
	} else {
		// Task Success Logic (Executor already set Status/Result)

		// Unblock downstream
		for _, downstreamID := range s.graph[taskID] {
			s.inDegree[downstreamID]--
			if s.inDegree[downstreamID] == 0 {
				s.readyQueue <- downstreamID
			}
		}
	}

	return nil
}

func (s *DAGScheduler) cascadeSkip(failedTaskID string) {
	// BFS or DFS to mark all reachable nodes as Skipped
	queue := []string{failedTaskID}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr] {
			continue
		}
		visited[curr] = true

		for _, downstream := range s.graph[curr] {
			downTask := s.tasks[downstream]
			// Use safe accessors
			if downTask.GetStatus() == TaskStatusPending {
				downTask.SetSkipped(fmt.Sprintf("Skipped due to upstream failure in %s", curr))
				queue = append(queue, downstream)

				// Also treat as "Done" for scheduler accounting logic?
				// Yes, otherwise we deadlock.
			}
		}
	}
}
