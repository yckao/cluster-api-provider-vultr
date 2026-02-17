/*
Copyright 2020 The Kubernetes Authors.

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
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/util"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"
)

// GetBareMetalServer retrieves a bare metal server by its ID.
func (s *Service) GetBareMetalServer(serverID string) (*govultr.BareMetalServer, error) {
	if serverID == "" {
		s.scope.Info("VultrBareMetalMachine does not have a server id")
		return nil, nil
	}

	s.scope.Logger.V(2).Info("Looking for bare metal server by ID", "server-id", serverID)

	server, resp, err := s.scope.BareMetalServers.Get(s.ctx, serverID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get bare metal server with ID %q", serverID)
	}

	return server, nil
}

func (s *Service) CreateBareMetalServer(machineScope *scope.BareMetalMachineScope) (*govultr.BareMetalServer, error) {
	s.scope.V(2).Info("Creating a bare metal server for a machine")

	s.scope.V(2).Info("Retrieving bootstrap data")
	bootstrapData, err := machineScope.GetBootstrapData()

	commands := []string{
		"ufw disable",
	}
	updatedBootstrapData := appendToUserDataCloudConfig(bootstrapData, commands)
	encodedBootstrapData := base64.StdEncoding.EncodeToString([]byte(updatedBootstrapData))

	if err != nil {
		log.Error(err, "Error getting bootstrap data for bare metal machine")
		return nil, errors.Wrap(err, "failed to retrieve bootstrap data")
	}
	s.scope.V(2).Info("Successfully retrieved bootstrap data")

	var sshKeyIDs []string //nolint:prealloc
	for _, sshKeyID := range machineScope.VultrBareMetalMachine.Spec.SSHKey {
		keys, err := s.GetSSHKey(sshKeyID)
		if err != nil {
			return nil, err
		}
		sshKeyIDs = append(sshKeyIDs, keys.ID)
	}
	clusterName := s.scope.Name()
	serverName := machineScope.Name()

	s.scope.V(2).Info("Preparing bare metal server creation request payload")
	bmCreateReq := &govultr.BareMetalCreate{
		Label:      serverName,
		Hostname:   serverName,
		Region:     machineScope.VultrBareMetalMachine.Spec.Region,
		Plan:       machineScope.VultrBareMetalMachine.Spec.PlanID,
		SSHKeyIDs:  sshKeyIDs,
		SnapshotID: machineScope.VultrBareMetalMachine.Spec.Snapshot,
		UserData:   encodedBootstrapData,
		EnableIPv6: util.Pointer(true),
	}

	s.scope.V(2).Info("Building bare metal server tags")
	bmCreateReq.Tags = infrav1.BuildTags(infrav1.BuildTagParams{
		ClusterName: clusterName,
		ClusterUID:  s.scope.UID(),
		Name:        serverName,
		Role:        machineScope.Role(),
	})
	s.scope.V(2).Info("Successfully built bare metal server tags")

	s.scope.V(2).Info("Creating bare metal server with Vultr API")
	server, _, err := s.scope.BareMetalServers.Create(s.ctx, bmCreateReq)
	if err != nil {
		log.Error(err, "Failed to create new bare metal server")
		return nil, errors.Wrap(err, "Failed to create new bare metal server")
	}
	s.scope.V(2).Info("Successfully created bare metal server", "server-id", server.ID)

	// Attach VPC after creation if VPCID is set
	if machineScope.VultrBareMetalMachine.Spec.VPCID != "" {
		s.scope.V(2).Info("Attaching VPC to bare metal server", "server-id", server.ID, "vpc-id", machineScope.VultrBareMetalMachine.Spec.VPCID)
		if err := s.scope.BareMetalServers.AttachVPC(s.ctx, server.ID, machineScope.VultrBareMetalMachine.Spec.VPCID); err != nil {
			s.scope.V(2).Info("Failed to attach VPC to bare metal server", "error", err)
			// Don't fail the creation if VPC attachment fails; the server is still created
		}
	}

	return server, nil
}

func (s *Service) DeleteBareMetalServer(id string) error {
	log.Info("Deleting bare metal server resources")
	s.scope.V(2).Info("Attempting to delete bare metal server", "server-id", id)
	if id == "" {
		s.scope.Info("Bare metal server does not have a server id")
		return errors.New("cannot delete bare metal server. server does not have a server id")
	}

	if err := s.scope.BareMetalServers.Delete(s.ctx, id); err != nil {
		return errors.Wrapf(err, "failed to delete bare metal server with id %q", id)
	}

	s.scope.V(2).Info("Deleted bare metal server", "server-id", id)
	return nil
}

// GetBareMetalServerAddress converts Vultr bare metal server IPs to corev1.NodeAddresses.
func (s *Service) GetBareMetalServerAddress(server *govultr.BareMetalServer) ([]corev1.NodeAddress, error) {
	addresses := []corev1.NodeAddress{}

	// Add public IPv4 address
	if server.MainIP != "" {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeExternalIP,
			Address: server.MainIP,
		})
	} else {
		s.scope.Info("No external IPv4 address found for the bare metal server", "server-id", server.ID)
	}

	// Check for VPC internal IP
	vpcInfoList, _, err := s.scope.BareMetalServers.ListVPCInfo(s.ctx, server.ID)
	if err == nil {
		for _, vpcInfo := range vpcInfoList {
			if vpcInfo.IPAddress != "" {
				addresses = append(addresses, corev1.NodeAddress{
					Type:    corev1.NodeInternalIP,
					Address: vpcInfo.IPAddress,
				})
				break
			}
		}
	}

	return addresses, nil
}

func (s *Service) AddBareMetalServerToVLB(vlbID, serverID string) error {
	for {
		currentVlb, _, err := s.scope.LoadBalancers.Get(context.TODO(), vlbID)
		if err != nil {
			return err
		}

		if currentVlb.Status != "active" {
			time.Sleep(10 * time.Second)
		} else {
			updateReq := govultr.LoadBalancerReq{}
			updateReq.Instances = append(currentVlb.Instances, serverID)
			err := s.scope.LoadBalancers.Update(context.TODO(), vlbID, &updateReq)
			return err
		}
	}
}
