/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package services

import (
	"github.com/vultr/govultr/v3"
)

// GetStartupScript returns a startup script by ID.
func (s *Service) GetStartupScript(id string) (*govultr.StartupScript, error) {
	if id == "" {
		return nil, nil
	}

	script, _, err := s.scope.StartupScripts.Get(s.ctx, id)
	if err != nil {
		return nil, err
	}

	return script, nil
}

// CreateStartupScript creates a new startup script.
func (s *Service) CreateStartupScript(req *govultr.StartupScriptReq) (*govultr.StartupScript, error) {
	script, _, err := s.scope.StartupScripts.Create(s.ctx, req)
	if err != nil {
		return nil, err
	}

	return script, nil
}

// DeleteStartupScript deletes a startup script by ID.
func (s *Service) DeleteStartupScript(id string) error {
	return s.scope.StartupScripts.Delete(s.ctx, id)
}
