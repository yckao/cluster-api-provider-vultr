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

// GetReservedIP returns a reserved IP by ID.
func (s *Service) GetReservedIP(id string) (*govultr.ReservedIP, error) {
	if id == "" {
		return nil, nil
	}

	rip, _, err := s.scope.ReservedIPs.Get(s.ctx, id)
	if err != nil {
		return nil, err
	}

	return rip, nil
}

// CreateReservedIP creates a new reserved IP.
func (s *Service) CreateReservedIP(req *govultr.ReservedIPReq) (*govultr.ReservedIP, error) {
	rip, _, err := s.scope.ReservedIPs.Create(s.ctx, req)
	if err != nil {
		return nil, err
	}

	return rip, nil
}

// DeleteReservedIP deletes a reserved IP by ID.
func (s *Service) DeleteReservedIP(id string) error {
	return s.scope.ReservedIPs.Delete(s.ctx, id)
}
