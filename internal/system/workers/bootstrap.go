package workers

import (
	"context"
	"fmt"

	schemaModel "github.com/wso2/identity-customer-data-service/internal/profile_schema/model"
	"github.com/wso2/identity-customer-data-service/internal/system/config"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	"github.com/wso2/identity-customer-data-service/internal/system/queue"

	// Built-in provider registration (memory)
	_ "github.com/wso2/identity-customer-data-service/internal/system/queue/providers/memory"
)

type WorkerManager struct {
	SchemaSyncWorker *SchemaSyncWorker
	ProfileWorker    *ProfileWorker
}

func Init(ctx context.Context, cfg config.Config) (*WorkerManager, error) {
	provider := cfg.Queue.Provider
	var providerCfg map[string]any

	// Memory = zero-config
	if provider == "" {
		provider = constants.MemoryQueueProvider
	}

	if provider != constants.MemoryQueueProvider {
		loaded, err := config.LoadProviderConfig(cfg.Queue.ProviderConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load provider config: %w", err)
		}
		providerCfg = loaded
	} else {
		providerCfg = map[string]any{}
	}

	// Schema sync queue
	schemaQ, err := queue.New[schemaModel.ProfileSchemaSync](provider, providerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema sync queue: %w", err)
	}

	// ProfileWorker queue
	unifyQ, err := queue.New[ProfileUnificationJob](provider, providerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create unification queue: %w", err)
	}

	m := &WorkerManager{
		SchemaSyncWorker: NewSchemaSyncWorker(schemaQ),
		ProfileWorker:    NewProfileWorker(unifyQ),
	}

	if err := m.SchemaSyncWorker.Start(ctx); err != nil {
		return nil, err
	}
	if err := m.ProfileWorker.Start(ctx); err != nil {
		return nil, err
	}

	return m, nil
}
