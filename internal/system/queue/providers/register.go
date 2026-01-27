package memory

import (
	schemaModel "github.com/wso2/identity-customer-data-service/internal/profile_schema/model"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	"github.com/wso2/identity-customer-data-service/internal/system/queue"
	"github.com/wso2/identity-customer-data-service/internal/system/workers"
)

func init() {
	// Memory provider: ZERO-CONFIG. Ignore cfg.
	queue.Register[schemaModel.ProfileSchemaSync](constants.MemoryQueueProvider,
		func(_ map[string]any) (queue.Queue[schemaModel.ProfileSchemaSync], error) {
			return New[schemaModel.ProfileSchemaSync](
				constants.SchemaSyncQueueName,
				constants.DefaultSchemaSyncQueueSize,
			), nil
		},
	)

	queue.Register[workers.ProfileUnificationJob](constants.MemoryQueueProvider,
		func(_ map[string]any) (queue.Queue[workers.ProfileUnificationJob], error) {
			return New[workers.ProfileUnificationJob](
				constants.UnificationQueueName,
				constants.DefaultUnificationQueueSize,
			), nil
		},
	)
}
