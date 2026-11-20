package runtime

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"kv-iot/datacenter/protocol"
)

type PollGroup struct {
	Name     string                  `json:"name"`
	Interval time.Duration           `json:"interval"`
	Points   []protocol.PointBinding `json:"points"`
}

type schedulerEntry struct {
	group PollGroup
	next  time.Time
}

// Scheduler tracks poll groups. The worker currently uses a single channel
// interval; this type provides the extension point for grouped polling.
type Scheduler struct {
	mu      sync.Mutex
	entries map[string]schedulerEntry
}

func NewScheduler() *Scheduler {
	return &Scheduler{entries: make(map[string]schedulerEntry)}
}

func (s *Scheduler) Upsert(now time.Time, group PollGroup) error {
	name := strings.TrimSpace(group.Name)
	if name == "" {
		return fmt.Errorf("poll group name is required")
	}
	if group.Interval <= 0 {
		return fmt.Errorf("poll group interval must be positive")
	}
	for index, point := range group.Points {
		if err := point.Validate(); err != nil {
			return fmt.Errorf("poll group point %d: %w", index, err)
		}
	}
	if now.IsZero() {
		now = time.Now()
	}
	s.mu.Lock()
	s.entries[name] = schedulerEntry{
		group: clonePollGroup(group),
		next:  now.Add(group.Interval),
	}
	s.mu.Unlock()
	return nil
}

// Due returns groups whose next run time has arrived and schedules the next
// run relative to now.
func (s *Scheduler) Due(now time.Time) []PollGroup {
	if now.IsZero() {
		now = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	due := make([]PollGroup, 0)
	for name, entry := range s.entries {
		if now.Before(entry.next) {
			continue
		}
		due = append(due, clonePollGroup(entry.group))
		entry.next = now.Add(entry.group.Interval)
		s.entries[name] = entry
	}
	sort.Slice(due, func(i, j int) bool {
		return due[i].Name < due[j].Name
	})
	return due
}

func (s *Scheduler) Remove(name string) {
	s.mu.Lock()
	delete(s.entries, strings.TrimSpace(name))
	s.mu.Unlock()
}

func (s *Scheduler) List() []PollGroup {
	s.mu.Lock()
	defer s.mu.Unlock()
	groups := make([]PollGroup, 0, len(s.entries))
	for _, entry := range s.entries {
		groups = append(groups, clonePollGroup(entry.group))
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})
	return groups
}

func clonePollGroup(group PollGroup) PollGroup {
	group.Points = clonePointBindings(group.Points)
	return group
}
