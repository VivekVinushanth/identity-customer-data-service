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

package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	adminConfigService "github.com/wso2/identity-customer-data-service/internal/admin_config/service"
	"github.com/wso2/identity-customer-data-service/internal/notification_template/model"
	"github.com/wso2/identity-customer-data-service/internal/notification_template/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
	"github.com/wso2/identity-customer-data-service/internal/system/security"
	"github.com/wso2/identity-customer-data-service/internal/system/utils"
)

type NotificationTemplateHandler struct{}

func NewNotificationTemplateHandler() *NotificationTemplateHandler {

	return &NotificationTemplateHandler{}
}

// AddNotificationTemplate handles POST /notification-templates.
func (h *NotificationTemplateHandler) AddNotificationTemplate(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "notification_templates:create"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	var req model.NotificationTemplateRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		clientError := errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.BAD_REQUEST.Code,
			Message:     errors2.BAD_REQUEST.Message,
			Description: utils.HandleDecodeError(err, "notification template"),
		}, http.StatusBadRequest)
		utils.WriteErrorResponse(w, clientError)
		return
	}

	now := time.Now().UTC()
	template := model.NotificationTemplate{
		TemplateId: uuid.New().String(),
		OrgHandle:  orgHandle,
		Channel:    req.Channel,
		Name:       req.Name,
		Subject:    req.Subject,
		Body:       req.Body,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	templateService := provider.NewNotificationTemplateProvider().GetNotificationTemplateService()
	if err := templateService.AddNotificationTemplate(template); err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusCreated, template, constants.NotificationTemplateResource)
}

// GetNotificationTemplates handles GET /notification-templates.
func (h *NotificationTemplateHandler) GetNotificationTemplates(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "notification_templates:view"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	templateService := provider.NewNotificationTemplateProvider().GetNotificationTemplateService()
	templates, err := templateService.GetNotificationTemplates(orgHandle)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, templates, constants.NotificationTemplateResource)
}

// GetNotificationTemplate handles GET /notification-templates/{templateId}.
func (h *NotificationTemplateHandler) GetNotificationTemplate(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "notification_templates:view"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	templateId := r.PathValue("templateId")
	if templateId == "" {
		utils.HandleError(w, notFoundError())
		return
	}

	templateService := provider.NewNotificationTemplateProvider().GetNotificationTemplateService()
	template, err := templateService.GetNotificationTemplate(templateId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, template, constants.NotificationTemplateResource)
}

// PatchNotificationTemplate handles PATCH /notification-templates/{templateId}.
func (h *NotificationTemplateHandler) PatchNotificationTemplate(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "notification_templates:update"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	templateId := r.PathValue("templateId")
	if templateId == "" {
		utils.HandleError(w, notFoundError())
		return
	}

	var patch model.NotificationTemplateUpdateRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&patch); err != nil {
		clientError := errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.BAD_REQUEST.Code,
			Message:     errors2.BAD_REQUEST.Message,
			Description: utils.HandleDecodeError(err, "notification template"),
		}, http.StatusBadRequest)
		utils.WriteErrorResponse(w, clientError)
		return
	}

	templateService := provider.NewNotificationTemplateProvider().GetNotificationTemplateService()
	template, err := templateService.GetNotificationTemplate(templateId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}

	if patch.Name != nil {
		template.Name = *patch.Name
	}
	if patch.Subject != nil {
		template.Subject = *patch.Subject
	}
	if patch.Body != nil {
		template.Body = *patch.Body
	}

	if err := templateService.UpdateNotificationTemplate(templateId, *template); err != nil {
		utils.HandleError(w, err)
		return
	}

	updated, err := templateService.GetNotificationTemplate(templateId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, updated, constants.NotificationTemplateResource)
}

// DeleteNotificationTemplate handles DELETE /notification-templates/{templateId}.
func (h *NotificationTemplateHandler) DeleteNotificationTemplate(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "notification_templates:delete"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	templateId := r.PathValue("templateId")
	if templateId == "" {
		utils.HandleError(w, notFoundError())
		return
	}

	templateService := provider.NewNotificationTemplateProvider().GetNotificationTemplateService()
	if err := templateService.DeleteNotificationTemplate(templateId); err != nil {
		utils.HandleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isCDSEnabled(orgHandle string) bool {
	return adminConfigService.GetAdminConfigService().IsCDSEnabled(orgHandle)
}

func cdsNotEnabledError() *errors2.ClientError {
	return errors2.NewClientError(errors2.ErrorMessage{
		Code:        errors2.CDS_NOT_ENABLED.Code,
		Message:     errors2.CDS_NOT_ENABLED.Message,
		Description: errors2.CDS_NOT_ENABLED.Description,
	}, http.StatusBadRequest)
}

func notFoundError() *errors2.ClientError {
	return errors2.NewClientError(errors2.ErrorMessage{
		Code:        errors2.NOTIFICATION_TEMPLATE_NOT_FOUND.Code,
		Message:     errors2.NOTIFICATION_TEMPLATE_NOT_FOUND.Message,
		Description: "Invalid path for notification template retrieval",
	}, http.StatusNotFound)
}
