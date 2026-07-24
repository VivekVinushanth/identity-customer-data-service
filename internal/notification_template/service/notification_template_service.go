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

package service

import (
	"fmt"
	"net/http"

	"github.com/wso2/identity-customer-data-service/internal/notification_template/model"
	"github.com/wso2/identity-customer-data-service/internal/notification_template/store"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
)

type NotificationTemplateServiceInterface interface {
	AddNotificationTemplate(template model.NotificationTemplate) error
	GetNotificationTemplates(orgHandle string) ([]model.NotificationTemplate, error)
	GetNotificationTemplate(templateId string) (*model.NotificationTemplate, error)
	UpdateNotificationTemplate(templateId string, template model.NotificationTemplate) error
	DeleteNotificationTemplate(templateId string) error
}

// NotificationTemplateService is the default implementation of NotificationTemplateServiceInterface.
type NotificationTemplateService struct{}

// GetNotificationTemplateService creates a new instance of NotificationTemplateService.
func GetNotificationTemplateService() NotificationTemplateServiceInterface {

	return &NotificationTemplateService{}
}

// AddNotificationTemplate validates and persists a new template.
func (s *NotificationTemplateService) AddNotificationTemplate(template model.NotificationTemplate) error {

	if err := validateTemplate(template); err != nil {
		return err
	}

	existing, err := store.GetNotificationTemplates(template.OrgHandle)
	if err != nil {
		return err
	}
	for _, t := range existing {
		if t.Channel == template.Channel && t.Name == template.Name {
			return errors2.NewClientError(errors2.ErrorMessage{
				Code:    errors2.NOTIFICATION_TEMPLATE_ALREADY_EXISTS.Code,
				Message: errors2.NOTIFICATION_TEMPLATE_ALREADY_EXISTS.Message,
				Description: fmt.Sprintf(
					"A %s template named %q already exists", template.Channel, template.Name),
			}, http.StatusConflict)
		}
	}

	return store.AddNotificationTemplate(template)
}

// GetNotificationTemplates fetches all templates for an org.
func (s *NotificationTemplateService) GetNotificationTemplates(orgHandle string) ([]model.NotificationTemplate, error) {
	return store.GetNotificationTemplates(orgHandle)
}

// GetNotificationTemplate fetches a specific template.
func (s *NotificationTemplateService) GetNotificationTemplate(templateId string) (*model.NotificationTemplate, error) {

	template, err := store.GetNotificationTemplate(templateId)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.NOTIFICATION_TEMPLATE_NOT_FOUND.Code,
			Message:     errors2.NOTIFICATION_TEMPLATE_NOT_FOUND.Message,
			Description: fmt.Sprintf("Notification template: '%s' not found", templateId),
		}, http.StatusNotFound)
	}
	return template, nil
}

// UpdateNotificationTemplate replaces the mutable fields of an existing template.
func (s *NotificationTemplateService) UpdateNotificationTemplate(templateId string, template model.NotificationTemplate) error {

	if err := validateTemplate(template); err != nil {
		return err
	}
	return store.UpdateNotificationTemplate(templateId, template)
}

// DeleteNotificationTemplate removes a template.
func (s *NotificationTemplateService) DeleteNotificationTemplate(templateId string) error {
	return store.DeleteNotificationTemplate(templateId)
}

func validateTemplate(template model.NotificationTemplate) error {

	if !constants.AllowedNotificationChannels[template.Channel] {
		return errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.NOTIFICATION_TEMPLATE_VALIDATION.Code,
			Message:     errors2.NOTIFICATION_TEMPLATE_VALIDATION.Message,
			Description: fmt.Sprintf("channel %q is not supported (expected \"email\" or \"sms\")", template.Channel),
		}, http.StatusBadRequest)
	}
	if template.Name == "" {
		return errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.NOTIFICATION_TEMPLATE_VALIDATION.Code,
			Message:     errors2.NOTIFICATION_TEMPLATE_VALIDATION.Message,
			Description: "name is required",
		}, http.StatusBadRequest)
	}
	if template.Body == "" {
		return errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.NOTIFICATION_TEMPLATE_VALIDATION.Code,
			Message:     errors2.NOTIFICATION_TEMPLATE_VALIDATION.Message,
			Description: "body is required",
		}, http.StatusBadRequest)
	}
	return nil
}
