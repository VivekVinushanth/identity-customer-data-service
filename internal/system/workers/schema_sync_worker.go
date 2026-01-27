/*
 * Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package workers

import (
	"context"
	"fmt"

	schemaModel "github.com/wso2/identity-customer-data-service/internal/profile_schema/model"
	schemaProvider "github.com/wso2/identity-customer-data-service/internal/profile_schema/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/log"
	"github.com/wso2/identity-customer-data-service/internal/system/queue"
)

type SchemaSyncWorker struct {
	q queue.Queue[schemaModel.ProfileSchemaSync]
}

func NewSchemaSyncWorker(q queue.Queue[schemaModel.ProfileSchemaSync]) *SchemaSyncWorker {
	return &SchemaSyncWorker{q: q}
}

func (w *SchemaSyncWorker) Start(ctx context.Context) error {
	return w.q.Start(ctx, func(ctx context.Context, job schemaModel.ProfileSchemaSync) error {
		logger := log.GetLogger()
		logger.Info(fmt.Sprintf("Processing schema sync job for tenant: %s, event: %s", job.OrgId, job.Event))

		svc := schemaProvider.NewProfileSchemaProvider().GetProfileSchemaService()
		if err := svc.SyncProfileSchema(job.OrgId); err != nil {
			logger.Error(fmt.Sprintf("Failed to sync profile schema for tenant: %s", job.OrgId), log.Error(err))
			return err
		}

		logger.Info(fmt.Sprintf("Profile schema sync completed successfully for tenant: %s", job.OrgId))
		return nil
	})
}

func (w *SchemaSyncWorker) Enqueue(ctx context.Context, job schemaModel.ProfileSchemaSync) error {
	return w.q.Enqueue(ctx, job)
}
