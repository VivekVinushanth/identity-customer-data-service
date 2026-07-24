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

package services

import (
	"net/http"
	"strings"

	"github.com/wso2/identity-customer-data-service/internal/notification_template/handler"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
)

type NotificationTemplateService struct {
	notificationTemplateHandler *handler.NotificationTemplateHandler
	mux                         *http.ServeMux
}

func NewNotificationTemplateService(mux *http.ServeMux) *NotificationTemplateService {
	s := &NotificationTemplateService{
		notificationTemplateHandler: handler.NewNotificationTemplateHandler(),
		mux:                         mux,
	}

	const base = constants.ApiBasePath + "/v1"
	s.mux.HandleFunc("POST "+base+"/notification-templates", s.notificationTemplateHandler.AddNotificationTemplate)
	s.mux.HandleFunc("GET "+base+"/notification-templates", s.notificationTemplateHandler.GetNotificationTemplates)
	s.mux.HandleFunc("GET "+base+"/notification-templates/{templateId}", s.notificationTemplateHandler.GetNotificationTemplate)
	s.mux.HandleFunc("PATCH "+base+"/notification-templates/{templateId}", s.notificationTemplateHandler.PatchNotificationTemplate)
	s.mux.HandleFunc("DELETE "+base+"/notification-templates/{templateId}", s.notificationTemplateHandler.DeleteNotificationTemplate)

	return s
}

// Route handles all tenant-aware notification template endpoints
func (s *NotificationTemplateService) Route(w http.ResponseWriter, r *http.Request) {
	if trimmed := strings.TrimSuffix(r.URL.Path, "/"); trimmed != "" {
		r.URL.Path = trimmed
	}
	s.mux.ServeHTTP(w, r)
}
