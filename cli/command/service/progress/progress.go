package progress

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/yuyangjack/moby/api/types"
	"github.com/yuyangjack/moby/api/types/filters"
	"github.com/yuyangjack/moby/api/types/swarm"
	"github.com/yuyangjack/moby/client"
	"github.com/yuyangjack/moby/pkg/progress"
	"github.com/yuyangjack/moby/pkg/streamformatter"
	"github.com/yuyangjack/moby/pkg/stringid"
)

var (
	numberedStates = map[swarm.TaskState]int64{
		swarm.TaskStateNew:       1,
		swarm.TaskStateAllocated: 2,
		swarm.TaskStatePending:   3,
		swarm.TaskStateAssigned:  4,
		swarm.TaskStateAccepted:  5,
		swarm.TaskStatePreparing: 6,
		swarm.TaskStateReady:     7,
		swarm.TaskStateStarting:  8,
		swarm.TaskStateRunning:   9,

		// The following states are not actually shown in progress
		// output, but are used internally for ordering.
		swarm.TaskStateComplete: 10,
		swarm.TaskStateShutdown: 11,
		swarm.TaskStateFailed:   12,
		swarm.TaskStateRejected: 13,
	}

	longestState int
)

const (
	maxProgress     = 9
	maxProgressBars = 20
)

type progressUpdater interface {
	update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error)
}

func init() {
	for state := range numberedStates {
		if !terminalState(state) && len(state) > longestState {
			longestState = len(state)
		}
	}
}

func terminalState(state swarm.TaskState) bool {
	return numberedStates[state] > numberedStates[swarm.TaskStateRunning]
}

func stateToProgress(state swarm.TaskState, rollback bool) int64 {
	if !rollback {
		return numberedStates[state]
	}
	return numberedStates[swarm.TaskStateRunning] - numberedStates[state]
}

// ServiceProgress outputs progress information for convergence of a service.
// nolint: gocyclo
func ServiceProgress(ctx context.Context, client client.APIClient, serviceID string, progressWriter io.WriteCloser) error {
	defer progressWriter.Close()

	progressOut := streamformatter.NewJSONProgressOutput(progressWriter, false)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	defer signal.Stop(sigint)

	taskFilter := filters.NewArgs()
	taskFilter.Add("service", serviceID)
	taskFilter.Add("_up-to-date", "true")

	getUpToDateTasks := func() ([]swarm.Task, error) {
		return client.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
	}

	var (
		updater     progressUpdater
		converged   bool
		convergedAt time.Time
		monitor     = 5 * time.Second
		rollback    bool
	)

	for {
		service, _, err := client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
		if err != nil {
			return err
		}

		if service.Spec.UpdateConfig != nil && service.Spec.UpdateConfig.Monitor != 0 {
			monitor = service.Spec.UpdateConfig.Monitor
		}

		if updater == nil {
			updater, err = initializeUpdater(service, progressOut)
			if err != nil {
				return err
			}
		}

		if service.UpdateStatus != nil {
			switch service.UpdateStatus.State {
			case swarm.UpdateStateUpdating:
				rollback = false
			case swarm.UpdateStateCompleted:
				if !converged {
					return nil
				}
			case swarm.UpdateStatePaused:
				return fmt.Errorf("service update paused: %s", service.UpdateStatus.Message)
			case swarm.UpdateStateRollbackStarted:
				if !rollback && service.UpdateStatus.Message != "" {
					progressOut.WriteProgress(progress.Progress{
						ID:     "rollback",
						Action: service.UpdateStatus.Message,
					})
				}
				rollback = true
			case swarm.UpdateStateRollbackPaused:
				return fmt.Errorf("service rollback paused: %s", service.UpdateStatus.Message)
			case swarm.UpdateStateRollbackCompleted:
				if !converged {
					return fmt.Errorf("service rolled back: %s", service.UpdateStatus.Message)
				}
			}
		}
		if converged && time.Since(convergedAt) >= monitor {
			progressOut.WriteProgress(progress.Progress{
				ID:     "verify",
				Action: "Service converged",
			})

			return nil
		}

		tasks, err := getUpToDateTasks()
		if err != nil {
			return err
		}

		activeNodes, err := getActiveNodes(ctx, client)
		if err != nil {
			return err
		}

		converged, err = updater.update(service, tasks, activeNodes, rollback)
		if err != nil {
			return err
		}
		if converged {
			if convergedAt.IsZero() {
				convergedAt = time.Now()
			}
			wait := monitor - time.Since(convergedAt)
			if wait >= 0 {
				progressOut.WriteProgress(progress.Progress{
					// Ideally this would have no ID, but
					// the progress rendering code behaves
					// poorly on an "action" with no ID. It
					// returns the cursor to the beginning
					// of the line, so the first character
					// may be difficult to read. Then the
					// output is overwritten by the shell
					// prompt when the command finishes.
					ID:     "verify",
					Action: fmt.Sprintf("Waiting %d seconds to verify that tasks are stable...", wait/time.Second+1),
				})
			}
		} else {
			if !convergedAt.IsZero() {
				progressOut.WriteProgress(progress.Progress{
					ID:     "verify",
					Action: "Detected task failure",
				})
			}
			convergedAt = time.Time{}
		}

		select {
		case <-time.After(200 * time.Millisecond):
		case <-sigint:
			if !converged {
				progress.Message(progressOut, "", "Operation continuing in background.")
				progress.Messagef(progressOut, "", "Use `docker service ps %s` to check progress.", serviceID)
			}
			return nil
		}
	}
}

func getActiveNodes(ctx context.Context, client client.APIClient) (map[string]struct{}, error) {
	nodes, err := client.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}

	activeNodes := make(map[string]struct{})
	for _, n := range nodes {
		if n.Status.State != swarm.NodeStateDown {
			activeNodes[n.ID] = struct{}{}
		}
	}
	return activeNodes, nil
}

func initializeUpdater(service swarm.Service, progressOut progress.Output) (progressUpdater, error) {
	if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
		return &replicatedProgressUpdater{
			progressOut: progressOut,
		}, nil
	}
	if service.Spec.Mode.Global != nil {
		return &globalProgressUpdater{
			progressOut: progressOut,
		}, nil
	}
	return nil, errors.New("unrecognized service mode")
}

func writeOverallProgress(progressOut progress.Output, numerator, denominator int, rollback bool) {
	if rollback {
		progressOut.WriteProgress(progress.Progress{
			ID:     "overall progress",
			Action: fmt.Sprintf("rolling back update: %d out of %d tasks", numerator, denominator),
		})
		return
	}
	progressOut.WriteProgress(progress.Progress{
		ID:     "overall progress",
		Action: fmt.Sprintf("%d out of %d tasks", numerator, denominator),
	})
}

func truncError(errMsg string) string {
	// Remove newlines from the error, which corrupt the output.
	errMsg = strings.Replace(errMsg, "\n", " ", -1)

	// Limit the length to 75 characters, so that even on narrow terminals
	// this will not overflow to the next line.
	if len(errMsg) > 75 {
		errMsg = errMsg[:74] + "…"
	}
	return errMsg
}

type replicatedProgressUpdater struct {
	progressOut progress.Output

	// used for mapping slots to a contiguous space
	// this also causes progress bars to appear in order
	slotMap map[int]int

	initialized bool
	done        bool
}

func (u *replicatedProgressUpdater) update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error) {
	if service.Spec.Mode.Replicated == nil || service.Spec.Mode.Replicated.Replicas == nil {
		return false, errors.New("no replica count")
	}
	replicas := *service.Spec.Mode.Replicated.Replicas

	if !u.initialized {
		u.slotMap = make(map[int]int)

		// Draw progress bars in order
		writeOverallProgress(u.progressOut, 0, int(replicas), rollback)

		if replicas <= maxProgressBars {
			for i := uint64(1); i <= replicas; i++ {
				progress.Update(u.progressOut, fmt.Sprintf("%d/%d", i, replicas), " ")
			}
		}
		u.initialized = true
	}

	tasksBySlot := u.tasksBySlot(tasks, activeNodes)

	// If we had reached a converged state, check if we are still converged.
	if u.done {
		for _, task := range tasksBySlot {
			if task.Status.State != swarm.TaskStateRunning {
				u.done = false
				break
			}
		}
	}

	running := uint64(0)

	for _, task := range tasksBySlot {
		mappedSlot := u.slotMap[task.Slot]
		if mappedSlot == 0 {
			mappedSlot = len(u.slotMap) + 1
			u.slotMap[task.Slot] = mappedSlot
		}

		if !terminalState(task.DesiredState) && task.Status.State == swarm.TaskStateRunning {
			running++
		}

		u.writeTaskProgress(task, mappedSlot, replicas, rollback)
	}

	if !u.done {
		writeOverallProgress(u.progressOut, int(running), int(replicas), rollback)

		if running == replicas {
			u.done = true
		}
	}

	return running == replicas, nil
}

func (u *replicatedProgressUpdater) tasksBySlot(tasks []swarm.Task, activeNodes map[string]struct{}) map[int]swarm.Task {
	// If there are multiple tasks with the same slot number, favor the one
	// with the *lowest* desired state. This can happen in restart
	// scenarios.
	tasksBySlot := make(map[int]swarm.Task)
	for _, task := range tasks {
		if numberedStates[task.DesiredState] == 0 || numberedStates[task.Status.State] == 0 {
			continue
		}
		if existingTask, ok := tasksBySlot[task.Slot]; ok {
			if numberedStates[existingTask.DesiredState] < numberedStates[task.DesiredState] {
				continue
			}
			// If the desired states match, observed state breaks
			// ties. This can happen with the "start first" service
			// update mode.
			if numberedStates[existingTask.DesiredState] == numberedStates[task.DesiredState] &&
				numberedStates[existingTask.Status.State] <= numberedStates[task.Status.State] {
				continue
			}
		}
		if task.NodeID != "" {
			if _, nodeActive := activeNodes[task.NodeID]; !nodeActive {
				continue
			}
		}
		tasksBySlot[task.Slot] = task
	}

	return tasksBySlot
}

func (u *replicatedProgressUpdater) writeTaskProgress(task swarm.Task, mappedSlot int, replicas uint64, rollback bool) {
	if u.done || replicas > maxProgressBars || uint64(mappedSlot) > replicas {
		return
	}

	if task.Status.Err != "" {
		u.progressOut.WriteProgress(progress.Progress{
			ID:     fmt.Sprintf("%d/%d", mappedSlot, replicas),
			Action: truncError(task.Status.Err),
		})
		return
	}

	if !terminalState(task.DesiredState) && !terminalState(task.Status.State) {
		u.progressOut.WriteProgress(progress.Progress{
			ID:         fmt.Sprintf("%d/%d", mappedSlot, replicas),
			Action:     fmt.Sprintf("%-[1]*s", longestState, task.Status.State),
			Current:    stateToProgress(task.Status.State, rollback),
			Total:      maxProgress,
			HideCounts: true,
		})
	}
}

type globalProgressUpdater struct {
	progressOut progress.Output

	initialized bool
	done        bool
}

func (u *globalProgressUpdater) update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error) {
	tasksByNode := u.tasksByNode(tasks)

	// We don't have perfect knowledge of how many nodes meet the
	// constraints for this service. But the orchestrator creates tasks
	// for all eligible nodes at the same time, so we should see all those
	// nodes represented among the up-to-date tasks.
	nodeCount := len(tasksByNode)

	if !u.initialized {
		if nodeCount == 0 {
			// Two possibilities: either the orchestrator hasn't created
			// the tasks yet, or the service doesn't meet constraints for
			// any node. Either way, we wait.
			u.progressOut.WriteProgress(progress.Progress{
				ID:     "overall progress",
				Action: "waiting for new tasks",
			})
			return false, nil
		}

		writeOverallProgress(u.progressOut, 0, nodeCount, rollback)
		u.initialized = true
	}

	// If we had reached a converged state, check if we are still converged.
	if u.done {
		for _, task := range tasksByNode {
			if task.Status.State != swarm.TaskStateRunning {
				u.done = false
				break
			}
		}
	}

	running := 0

	for _, task := range tasksByNode {
		if _, nodeActive := activeNodes[task.NodeID]; nodeActive {
			if !terminalState(task.DesiredState) && task.Status.State == swarm.TaskStateRunning {
				running++
			}

			u.writeTaskProgress(task, nodeCount, rollback)
		}
	}

	if !u.done {
		writeOverallProgress(u.progressOut, running, nodeCount, rollback)

		if running == nodeCount {
			u.done = true
		}
	}

	return running == nodeCount, nil
}

func (u *globalProgressUpdater) tasksByNode(tasks []swarm.Task) map[string]swarm.Task {
	// If there are multiple tasks with the same node ID, favor the one
	// with the *lowest* desired state. This can happen in restart
	// scenarios.
	tasksByNode := make(map[string]swarm.Task)
	for _, task := range tasks {
		if numberedStates[task.DesiredState] == 0 || numberedStates[task.Status.State] == 0 {
			continue
		}
		if existingTask, ok := tasksByNode[task.NodeID]; ok {
			if numberedStates[existingTask.DesiredState] < numberedStates[task.DesiredState] {
				continue
			}

			// If the desired states match, observed state breaks
			// ties. This can happen with the "start first" service
			// update mode.
			if numberedStates[existingTask.DesiredState] == numberedStates[task.DesiredState] &&
				numberedStates[existingTask.Status.State] <= numberedStates[task.Status.State] {
				continue
			}

		}
		tasksByNode[task.NodeID] = task
	}

	return tasksByNode
}

func (u *globalProgressUpdater) writeTaskProgress(task swarm.Task, nodeCount int, rollback bool) {
	if u.done || nodeCount > maxProgressBars {
		return
	}

	if task.Status.Err != "" {
		u.progressOut.WriteProgress(progress.Progress{
			ID:     stringid.TruncateID(task.NodeID),
			Action: truncError(task.Status.Err),
		})
		return
	}

	if !terminalState(task.DesiredState) && !terminalState(task.Status.State) {
		u.progressOut.WriteProgress(progress.Progress{
			ID:         stringid.TruncateID(task.NodeID),
			Action:     fmt.Sprintf("%-[1]*s", longestState, task.Status.State),
			Current:    stateToProgress(task.Status.State, rollback),
			Total:      maxProgress,
			HideCounts: true,
		})
	}
}
