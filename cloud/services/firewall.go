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

// GetFirewallGroup returns a firewall group by ID.
func (s *Service) GetFirewallGroup(id string) (*govultr.FirewallGroup, error) {
	if id == "" {
		return nil, nil
	}

	fg, _, err := s.scope.FirewallGroups.Get(s.ctx, id)
	if err != nil {
		return nil, err
	}

	return fg, nil
}

// CreateFirewallGroup creates a new firewall group.
func (s *Service) CreateFirewallGroup(req *govultr.FirewallGroupReq) (*govultr.FirewallGroup, error) {
	fg, _, err := s.scope.FirewallGroups.Create(s.ctx, req)
	if err != nil {
		return nil, err
	}

	return fg, nil
}

// DeleteFirewallGroup deletes a firewall group by ID.
func (s *Service) DeleteFirewallGroup(id string) error {
	return s.scope.FirewallGroups.Delete(s.ctx, id)
}

// ListFirewallRules lists all firewall rules in a group.
func (s *Service) ListFirewallRules(groupID string) ([]govultr.FirewallRule, error) {
	var allRules []govultr.FirewallRule
	options := &govultr.ListOptions{PerPage: 100}

	for {
		rules, meta, _, err := s.scope.FirewallRules.List(s.ctx, groupID, options)
		if err != nil {
			return nil, err
		}

		allRules = append(allRules, rules...)

		if meta.Links.Next == "" {
			break
		}

		options.Cursor = meta.Links.Next
	}

	return allRules, nil
}

// CreateFirewallRule creates a new firewall rule in a group.
func (s *Service) CreateFirewallRule(groupID string, req *govultr.FirewallRuleReq) (*govultr.FirewallRule, error) {
	rule, _, err := s.scope.FirewallRules.Create(s.ctx, groupID, req)
	if err != nil {
		return nil, err
	}

	return rule, nil
}

// DeleteFirewallRule deletes a firewall rule from a group.
func (s *Service) DeleteFirewallRule(groupID string, ruleNumber int) error {
	return s.scope.FirewallRules.Delete(s.ctx, groupID, ruleNumber)
}
