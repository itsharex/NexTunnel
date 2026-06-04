package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Scheduler manages multiple network paths and selects the optimal one.
type Scheduler struct {
	config     SchedulerConfig
	paths      sync.Map // pathID -> *Path
	activePath atomic.Value
	lockedID   atomic.Value // string (empty = not locked)
	lastSwitch atomic.Value // time.Time

	onPathChange func(oldPath, newPath *Path)

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewScheduler creates a new link scheduler.
func NewScheduler(cfg SchedulerConfig, opts ...SchedulerOption) *Scheduler {
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		logger: cfg.Logger,
	}
	s.lockedID.Store("")
	s.lastSwitch.Store(time.Time{})
	return s
}

// RegisterPath adds a new path to the scheduler.
func (s *Scheduler) RegisterPath(path *Path) {
	path.CreatedAt = time.Now()
	s.paths.Store(path.ID, path)
	s.logger.Info("path registered", "id", path.ID, "type", path.Type)

	if s.ActivePath() == nil {
		s.activePath.Store(path)
		s.logger.Info("initial active path set", "id", path.ID)
	}
}

// UnregisterPath removes a path from the scheduler.
func (s *Scheduler) UnregisterPath(pathID string) {
	s.paths.Delete(pathID)
	s.logger.Info("path unregistered", "id", pathID)

	if active := s.ActivePath(); active != nil && active.ID == pathID {
		s.activePath.Store((*Path)(nil))
		s.Evaluate()
	}
}

// ActivePath returns the currently active path, or nil.
func (s *Scheduler) ActivePath() *Path {
	v := s.activePath.Load()
	if v == nil {
		return nil
	}
	p, _ := v.(*Path)
	return p
}

// AllPaths returns all registered paths.
func (s *Scheduler) AllPaths() []*Path {
	var result []*Path
	s.paths.Range(func(_, value any) bool {
		result = append(result, value.(*Path))
		return true
	})
	return result
}

// LockPath manually locks the scheduler to a specific path.
func (s *Scheduler) LockPath(pathID string) error {
	v, ok := s.paths.Load(pathID)
	if !ok {
		return fmt.Errorf("path not found: %s", pathID)
	}
	path := v.(*Path)
	path.ManualLock = true
	s.lockedID.Store(pathID)
	s.activePath.Store(path)
	s.logger.Info("path locked", "id", pathID, "type", path.Type)
	return nil
}

// UnlockPath removes the manual path lock.
func (s *Scheduler) UnlockPath() {
	lockedID := s.lockedID.Load().(string)
	if lockedID != "" {
		if v, ok := s.paths.Load(lockedID); ok {
			v.(*Path).ManualLock = false
		}
	}
	s.lockedID.Store("")
	s.logger.Info("path unlocked")
	s.Evaluate()
}

// Evaluate re-evaluates all paths and returns the best one.
func (s *Scheduler) Evaluate() *Path {
	lockedID := s.lockedID.Load().(string)
	if lockedID != "" {
		if v, ok := s.paths.Load(lockedID); ok {
			return v.(*Path)
		}
	}

	paths := s.AllPaths()
	if len(paths) == 0 {
		return nil
	}

	scores := RankPaths(paths)
	best := SelectBestPath(scores)
	if best == nil {
		return nil
	}

	for _, p := range paths {
		if p.ID == best.PathID {
			return p
		}
	}
	return nil
}

// SwitchTo switches the active path to the given path ID.
func (s *Scheduler) SwitchTo(pathID string) error {
	v, ok := s.paths.Load(pathID)
	if !ok {
		return fmt.Errorf("path not found: %s", pathID)
	}

	newPath := v.(*Path)
	oldPath := s.ActivePath()

	if oldPath != nil && oldPath.ID == pathID {
		return nil
	}

	lastSwitch := s.lastSwitch.Load().(time.Time)
	if time.Since(lastSwitch) < s.config.SwitchCooldown {
		return fmt.Errorf("switch cooldown active")
	}

	s.activePath.Store(newPath)
	s.lastSwitch.Store(time.Now())

	s.logger.Info("path switched", "to", newPath.ID, "type", newPath.Type)

	if s.onPathChange != nil {
		s.onPathChange(oldPath, newPath)
	}
	return nil
}

// OnPathChange registers a callback for path changes.
func (s *Scheduler) OnPathChange(fn func(oldPath, newPath *Path)) {
	s.onPathChange = fn
}

// Start begins the periodic evaluation loop.
func (s *Scheduler) Start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)
	if !s.config.EnableAutoSwitch {
		return
	}
	go s.evalLoop()
	s.logger.Info("scheduler started", "eval_interval", s.config.EvalInterval)
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.cancel()
	s.logger.Info("scheduler stopped")
}

func (s *Scheduler) evalLoop() {
	ticker := time.NewTicker(s.config.EvalInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.autoEvaluate()
		}
	}
}

func (s *Scheduler) autoEvaluate() {
	lockedID := s.lockedID.Load().(string)
	if lockedID != "" {
		return
	}

	active := s.ActivePath()
	if active == nil {
		best := s.Evaluate()
		if best != nil {
			s.SwitchTo(best.ID)
		}
		return
	}

	if active.Prober != nil {
		active.Metrics = active.Prober.Metrics()
	}

	if active.Metrics.LossRate > s.config.LossThreshold {
		best := s.Evaluate()
		if best != nil && best.ID != active.ID {
			s.logger.Info("path degraded, switching",
				"loss", active.Metrics.LossRate,
				"to", best.ID)
			s.SwitchTo(best.ID)
		}
	}
}
