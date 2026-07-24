/*
 * Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/wso2/identity-customer-data-service/internal/notification_template/model"
	"github.com/wso2/identity-customer-data-service/internal/system/database/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/database/scripts"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
	"github.com/wso2/identity-customer-data-service/internal/system/log"
)

// AddNotificationTemplate persists a new notification template.
func AddNotificationTemplate(template model.NotificationTemplate) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		return dbError(errors2.ADD_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Failed to get database client for adding notification template: %s", template.Name), err)
	}
	defer dbClient.Close()

	query := scripts.InsertNotificationTemplate[provider.NewDBProvider().GetDBType()]
	_, err = dbClient.ExecuteQuery(query, template.TemplateId, template.OrgHandle, template.Channel, template.Name,
		nullableString(template.Subject), template.Body, template.CreatedAt, template.UpdatedAt)
	if err != nil {
		return dbError(errors2.ADD_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Error occurred while adding notification template: %s", template.Name), err)
	}

	logger.Info(fmt.Sprintf("Notification template '%s' added successfully", template.Name))
	return nil
}

// GetNotificationTemplates fetches all templates for an org, sorted by name.
func GetNotificationTemplates(orgHandle string) ([]model.NotificationTemplate, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return nil, dbError(errors2.GET_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Failed to get database client for fetching notification templates for organization: %s", orgHandle), err)
	}
	defer dbClient.Close()

	query := scripts.GetNotificationTemplates[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, orgHandle)
	if err != nil {
		return nil, dbError(errors2.GET_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Failed in fetching notification templates for organization: %s", orgHandle), err)
	}
	return rowsToTemplates(results), nil
}

// GetNotificationTemplate fetches a specific template by its Id.
func GetNotificationTemplate(templateId string) (*model.NotificationTemplate, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return nil, dbError(errors2.GET_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Failed to get database client for fetching notification template: %s", templateId), err)
	}
	defer dbClient.Close()

	query := scripts.GetNotificationTemplate[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, templateId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, dbError(errors2.GET_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Failed in fetching notification template with template_id: %s", templateId), err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	templates := rowsToTemplates(results)
	return &templates[0], nil
}

// UpdateNotificationTemplate replaces the mutable fields of an existing template.
func UpdateNotificationTemplate(templateId string, template model.NotificationTemplate) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return dbError(errors2.UPDATE_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Failed to get database client for updating notification template: %s", templateId), err)
	}
	defer dbClient.Close()

	query := scripts.UpdateNotificationTemplate[provider.NewDBProvider().GetDBType()]
	_, err = dbClient.ExecuteQuery(query, template.Name, nullableString(template.Subject), template.Body,
		time.Now().UTC(), templateId)
	if err != nil {
		return dbError(errors2.UPDATE_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Error occurred while updating notification template for template_id: %s", templateId), err)
	}
	return nil
}

// DeleteNotificationTemplate deletes a template by its Id.
func DeleteNotificationTemplate(templateId string) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return dbError(errors2.DELETE_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Failed to get database client for deleting notification template: %s", templateId), err)
	}
	defer dbClient.Close()

	query := scripts.DeleteNotificationTemplate[provider.NewDBProvider().GetDBType()]
	_, err = dbClient.ExecuteQuery(query, templateId)
	if err != nil {
		return dbError(errors2.DELETE_NOTIFICATION_TEMPLATE, fmt.Sprintf(
			"Failed to delete notification template: %s", templateId), err)
	}
	return nil
}

func rowsToTemplates(results []map[string]interface{}) []model.NotificationTemplate {

	templates := make([]model.NotificationTemplate, 0, len(results))
	for _, row := range results {
		var template model.NotificationTemplate
		template.TemplateId, _ = row["template_id"].(string)
		template.OrgHandle, _ = row["org_handle"].(string)
		template.Channel, _ = row["channel"].(string)
		template.Name, _ = row["name"].(string)
		template.Subject, _ = row["subject"].(string)
		template.Body, _ = row["body"].(string)
		if createdAt, ok := row["created_at"].(time.Time); ok {
			template.CreatedAt = createdAt
		}
		if updatedAt, ok := row["updated_at"].(time.Time); ok {
			template.UpdatedAt = updatedAt
		}
		templates = append(templates, template)
	}
	return templates
}

func dbError(base errors2.ErrorMessage, description string, cause error) error {
	log.GetLogger().Debug(description, log.Error(cause))
	return errors2.NewServerError(errors2.ErrorMessage{
		Code:        base.Code,
		Message:     base.Message,
		Description: description,
	}, cause)
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
