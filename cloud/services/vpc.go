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

// GetVPC returns a VPC by ID.
func (s *Service) GetVPC(id string) (*govultr.VPC, error) {
	if id == "" {
		return nil, nil
	}

	vpc, _, err := s.scope.VPCs.Get(s.ctx, id)
	if err != nil {
		return nil, err
	}

	return vpc, nil
}

// CreateVPC creates a new VPC.
func (s *Service) CreateVPC(req *govultr.VPCReq) (*govultr.VPC, error) {
	vpc, _, err := s.scope.VPCs.Create(s.ctx, req)
	if err != nil {
		return nil, err
	}

	return vpc, nil
}

// DeleteVPC deletes a VPC by ID.
func (s *Service) DeleteVPC(id string) error {
	return s.scope.VPCs.Delete(s.ctx, id)
}
