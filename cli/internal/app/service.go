// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/KyleDerZweite/basalt/internal/config"
	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
	"github.com/KyleDerZweite/basalt/internal/output"
	"github.com/KyleDerZweite/basalt/internal/walker"
)

// Service is the local product backend shared by the CLI and local clients.
type Service struct {
	version string
	dataDir string
	store   *Store
	broker  *eventBroker

	mu     sync.Mutex
	active map[string]context.CancelFunc
}

// NewService creates the shared backend service.
func NewService(version, dataDir string) (*Service, error) {
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}

	store, err := openStore(dataDir)
	if err != nil {
		return nil, err
	}

	return &Service{
		version: version,
		dataDir: dataDir,
		store:   store,
		broker:  newEventBroker(),
		active:  make(map[string]context.CancelFunc),
	}, nil
}

// Close releases service resources.
func (s *Service) Close() error {
	return s.store.Close()
}

// DataDir returns the service data directory.
func (s *Service) DataDir() string {
	return s.dataDir
}

// Version returns the current application version associated with the service.
func (s *Service) Version() string {
	return s.version
}

// GetSettings returns local product settings.
func (s *Service) GetSettings() (Settings, error) {
	return s.store.GetSettings()
}

// UpdateSettings persists local product settings.
func (s *Service) UpdateSettings(settings Settings) error {
	return s.store.SaveSettings(settings)
}

// StartScan starts a background scan and returns immediately.
func (s *Service) StartScan(_ context.Context, req ScanRequest) (*ScanRecord, error) {
	record, settings, err := s.prepareScan(req)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.setActive(record.ID, cancel)

	go func() {
		defer s.clearActive(record.ID)
		s.executeScan(ctx, record, settings)
	}()

	return record, nil
}

// RunScan executes a scan synchronously and persists it locally.
func (s *Service) RunScan(ctx context.Context, req ScanRequest) (*ScanRecord, error) {
	record, settings, err := s.prepareScan(req)
	if err != nil {
		return nil, err
	}

	s.executeScan(ctx, record, settings)
	return s.store.GetScan(record.ID)
}

// CancelScan cancels a running scan.
func (s *Service) CancelScan(id string) error {
	s.mu.Lock()
	cancel := s.active[id]
	s.mu.Unlock()
	if cancel == nil {
		return fmt.Errorf("scan %s is not running", id)
	}
	cancel()
	return nil
}

// ListScans returns recent persisted scans.
func (s *Service) ListScans(limit int) ([]*ScanRecord, error) {
	return s.store.ListScans(limit)
}

// GetScan returns a persisted scan by ID.
func (s *Service) GetScan(id string) (*ScanRecord, error) {
	return s.store.GetScan(id)
}

// GetEvents returns persisted scan events after a given sequence number.
func (s *Service) GetEvents(scanID string, afterSeq int64) ([]ScanEvent, error) {
	return s.store.ListEvents(scanID, afterSeq)
}

// Subscribe streams new events for a scan.
func (s *Service) Subscribe(scanID string) (<-chan ScanEvent, func()) {
	return s.broker.Subscribe(scanID)
}

// WriteExport writes a persisted scan export in the requested format.
func (s *Service) WriteExport(scanID, format string, out io.Writer) error {
	record, err := s.store.GetScan(scanID)
	if err != nil {
		return err
	}
	if record.Graph == nil {
		return fmt.Errorf("scan %s has no stored graph", scanID)
	}

	switch format {
	case "json":
		return output.WriteJSON(out, record.Graph)
	case "csv":
		return output.WriteCSV(out, record.Graph)
	default:
		return fmt.Errorf("unsupported export format %q", format)
	}
}

// ModuleHealth returns the effective module health under current settings.
func (s *Service) ModuleHealth(ctx context.Context, req ScanRequest) ([]ModuleStatus, error) {
	req.Normalize()

	settings, err := s.store.GetSettings()
	if err != nil {
		return nil, err
	}
	effective := mergeSettings(settings, req)
	cfg, err := s.loadConfig(req.ConfigPath)
	if err != nil {
		return nil, err
	}

	disabled := makeDisabledSet(effective.DisabledModules)
	registry := buildRegistry(cfg, disabled)
	client := httpclient.New(httpclient.WithTimeout(time.Duration(req.TimeoutSeconds) * time.Second))
	w := walker.New(graph.New(), registry, walker.WithClient(client), walker.WithTimeout(time.Duration(req.TimeoutSeconds)*time.Second))
	w.VerifyAll(ctx)

	health := healthFromWalker(w.HealthSummary())
	if effective.StrictMode {
		health = applyStrictHealth(health)
	}
	return health, nil
}

func (s *Service) prepareScan(req ScanRequest) (*ScanRecord, Settings, error) {
	req.Normalize()
	if len(req.Seeds) == 0 {
		return nil, Settings{}, fmt.Errorf("at least one seed is required")
	}

	settings, err := s.store.GetSettings()
	if err != nil {
		return nil, Settings{}, err
	}

	now := time.Now().UTC()
	record := &ScanRecord{
		ID:        uuid.New().String(),
		Status:    ScanStatusQueued,
		StartedAt: now,
		UpdatedAt: now,
		Seeds:     req.Seeds,
		Options:   req,
	}
	if err := s.store.CreateScan(record); err != nil {
		return nil, Settings{}, err
	}
	s.publishEvent(&ScanEvent{
		ScanID: record.ID,
		Time:   now,
		Type:   "scan_queued",
		Data: map[string]interface{}{
			"seed_count": len(req.Seeds),
		},
	})
	return record, settings, nil
}

func (s *Service) executeScan(ctx context.Context, record *ScanRecord, settings Settings) {
	effectiveSettings := mergeSettings(settings, record.Options)
	record.Status = ScanStatusVerifying
	record.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateScan(record); err != nil {
		s.failScan(record, err)
		return
	}
	s.publishEvent(&ScanEvent{ScanID: record.ID, Type: "scan_status", Time: record.UpdatedAt, Data: map[string]interface{}{"status": record.Status}})

	cfg, err := s.loadConfig(record.Options.ConfigPath)
	if err != nil {
		s.failScan(record, err)
		return
	}

	disabled := makeDisabledSet(effectiveSettings.DisabledModules)
	registry := buildRegistry(cfg, disabled)
	graphData := graph.New()
	graphData.Meta.Version = s.version
	graphData.Meta.ScanID = record.ID
	graphData.Meta.StartedAt = record.StartedAt
	graphData.Meta.Config = graph.Config{
		MaxPivotDepth: record.Options.Depth,
		Concurrency:   record.Options.Concurrency,
		TimeoutSecs:   record.Options.TimeoutSeconds,
	}
	for _, seed := range record.Seeds {
		graphData.Meta.InitialSeeds = append(graphData.Meta.InitialSeeds, graph.SeedRef{
			Value: seed.Value,
			Type:  seed.Type,
		})
	}

	client := httpclient.New(httpclient.WithTimeout(time.Duration(record.Options.TimeoutSeconds) * time.Second))
	w := walker.New(
		graphData,
		registry,
		walker.WithMaxDepth(record.Options.Depth),
		walker.WithConcurrency(record.Options.Concurrency),
		walker.WithTimeout(time.Duration(record.Options.TimeoutSeconds)*time.Second),
		walker.WithClient(client),
		walker.WithEventHandler(func(event walker.Event) {
			s.publishEvent(&ScanEvent{
				ScanID:  record.ID,
				Time:    event.Time,
				Type:    event.Type,
				Module:  event.Module,
				NodeID:  event.NodeID,
				EdgeID:  event.EdgeID,
				Message: event.Message,
				Data:    event.Data,
			})
		}),
	)

	w.VerifyAll(ctx)
	healthSummary := w.HealthSummary()
	record.Health = healthFromWalker(healthSummary)
	if effectiveSettings.StrictMode {
		healthSummary = applyStrictModuleHealth(healthSummary)
		record.Health = healthFromWalker(healthSummary)
		w.SetHealthSummary(healthSummary)
	}
	record.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateScan(record); err != nil {
		s.failScan(record, err)
		return
	}
	s.publishEvent(&ScanEvent{
		ScanID: record.ID,
		Time:   record.UpdatedAt,
		Type:   "verify_complete",
		Data: map[string]interface{}{
			"module_count": len(record.Health),
		},
	})

	record.Status = ScanStatusRunning
	record.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateScan(record); err != nil {
		s.failScan(record, err)
		return
	}
	s.publishEvent(&ScanEvent{ScanID: record.ID, Type: "scan_status", Time: record.UpdatedAt, Data: map[string]interface{}{"status": record.Status}})

	w.Run(ctx, record.Seeds)

	completedAt := time.Now().UTC()
	graphData.Meta.CompletedAt = completedAt
	graphData.Meta.DurationSecs = completedAt.Sub(record.StartedAt).Seconds()

	record.CompletedAt = &completedAt
	record.UpdatedAt = completedAt
	record.Graph = graphData
	nodes, edges := graphData.Collect()
	record.NodeCount = len(nodes)
	record.EdgeCount = len(edges)
	switch {
	case ctx.Err() != nil && (record.NodeCount > len(record.Seeds) || record.EdgeCount > 0):
		record.Status = ScanStatusPartial
	case ctx.Err() != nil:
		record.Status = ScanStatusCanceled
	default:
		record.Status = ScanStatusCompleted
	}

	if err := s.store.UpdateScan(record); err != nil {
		s.failScan(record, err)
		return
	}
	s.publishEvent(&ScanEvent{
		ScanID: record.ID,
		Time:   completedAt,
		Type:   "scan_finished",
		Data: map[string]interface{}{
			"status":     record.Status,
			"node_count": record.NodeCount,
			"edge_count": record.EdgeCount,
		},
	})
}

func (s *Service) failScan(record *ScanRecord, err error) {
	now := time.Now().UTC()
	record.Status = ScanStatusFailed
	record.ErrorMessage = err.Error()
	record.UpdatedAt = now
	record.CompletedAt = &now
	_ = s.store.UpdateScan(record)
	s.publishEvent(&ScanEvent{
		ScanID:  record.ID,
		Time:    now,
		Type:    "scan_failed",
		Message: err.Error(),
	})
}

func (s *Service) loadConfig(configPath string) (*config.Config, error) {
	if configPath == "" {
		configPath = DefaultConfigPath(s.dataDir)
	}
	return config.Load(configPath)
}

func (s *Service) publishEvent(event *ScanEvent) {
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	if err := s.store.AppendEvent(event); err == nil {
		s.broker.Publish(*event)
	}
}

func (s *Service) setActive(scanID string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[scanID] = cancel
}

func (s *Service) clearActive(scanID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, scanID)
}

func mergeSettings(settings Settings, req ScanRequest) Settings {
	settings = normalizeSettings(settings)
	if req.StrictMode {
		settings.StrictMode = true
	}
	settings.DisabledModules = dedupeStrings(append(settings.DisabledModules, req.DisabledModules...))
	return settings
}

func makeDisabledSet(disabled []string) map[string]struct{} {
	out := make(map[string]struct{}, len(disabled))
	for _, name := range disabled {
		out[strings.TrimSpace(name)] = struct{}{}
	}
	return out
}

func healthFromWalker(in []walker.ModuleHealth) []ModuleStatus {
	out := make([]ModuleStatus, 0, len(in))
	for _, item := range in {
		out = append(out, ModuleStatus{
			Name:        item.Module.Name(),
			Description: item.Module.Description(),
			Status:      item.Status.String(),
			Message:     item.Message,
		})
	}
	return out
}

func applyStrictHealth(in []ModuleStatus) []ModuleStatus {
	out := make([]ModuleStatus, len(in))
	copy(out, in)
	for i := range out {
		if out[i].Status == "degraded" {
			out[i].Status = "offline"
			if out[i].Message != "" {
				out[i].Message += " (disabled by strict mode)"
			} else {
				out[i].Message = "disabled by strict mode"
			}
		}
	}
	return out
}

func applyStrictModuleHealth(in []walker.ModuleHealth) []walker.ModuleHealth {
	out := make([]walker.ModuleHealth, len(in))
	copy(out, in)
	for i := range out {
		if out[i].Status.String() != "degraded" {
			continue
		}
		out[i].Status = modules.Offline
		if out[i].Message != "" {
			out[i].Message += " (disabled by strict mode)"
		} else {
			out[i].Message = "disabled by strict mode"
		}
	}
	return out
}

type eventBroker struct {
	mu   sync.Mutex
	subs map[string]map[chan ScanEvent]struct{}
}

func newEventBroker() *eventBroker {
	return &eventBroker{subs: make(map[string]map[chan ScanEvent]struct{})}
}

func (b *eventBroker) Publish(event ScanEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[event.ScanID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *eventBroker) Subscribe(scanID string) (<-chan ScanEvent, func()) {
	ch := make(chan ScanEvent, 32)

	b.mu.Lock()
	if b.subs[scanID] == nil {
		b.subs[scanID] = make(map[chan ScanEvent]struct{})
	}
	b.subs[scanID][ch] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[scanID][ch]; !ok {
			return
		}
		delete(b.subs[scanID], ch)
		close(ch)
		if len(b.subs[scanID]) == 0 {
			delete(b.subs, scanID)
		}
	}
	return ch, cancel
}
