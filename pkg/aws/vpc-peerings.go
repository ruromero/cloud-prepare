/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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
package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"
	"github.com/submariner-io/cloud-prepare/pkg/api"
)

const (
	attempts = 3
	waitTime = 10
)

func (ac *awsCloud) createAWSPeering(target *awsCloud, reporter api.Reporter) error {
	reporter.Started("Creating VPC Peering between %v/%v and %v/%v", ac.infraID, ac.region, target.infraID, target.region)

	err := ac.validatePeeringPrerequisites(target, reporter)
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to validate vpc peering prerequisites")
	}

	sourceVpcID, err := ac.getVpcID()
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to retrieve source VPC ID")
	}

	targetVpcID, err := target.getVpcID()
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to retrieve target VPC ID")
	}

	peering, err := ac.requestPeering(sourceVpcID, targetVpcID, target, reporter)
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to request VPC peering")
	}

	// Needed to retry the query to accept the peering in order to wait for AWS to create the peering resource in both environments
	reporter.Started("Trying again to Accept peering")
	acceptPeeringFn := func() error {
		return target.acceptPeering(peering.VpcPeeringConnectionId, reporter)
	}
	err = runWithRetries(acceptPeeringFn)
	if err != nil {
		return errors.Wrapf(err, "unable to accept VPC peering")
	}

	// Peering routes creation. It should create two routes (one per cluster) to forward
	// the traffic between clusters throught the peering object based on CIDR blocks
	reporter.Started("Trying again to Create routes for VPC Peering")
	createRoutesFn := func() error {
		return ac.createRoutesForPeering(target, sourceVpcID, targetVpcID, peering, reporter)
	}
	err = runWithRetries(createRoutesFn)
	if err != nil {
		return errors.Wrapf(err, "unable to create routes for VPC peering")
	}

	reporter.Succeeded("Created VPC Peering")
	return nil
}

func (ac *awsCloud) requestPeering(srcVpcID, targetVpcID string, target *awsCloud,
	reporter api.Reporter) (*types.VpcPeeringConnection, error) {
	reporter.Started("Requesting VPC Peering")

	input := &ec2.CreateVpcPeeringConnectionInput{
		VpcId:      &srcVpcID,
		PeerVpcId:  &targetVpcID,
		PeerRegion: &target.region,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpcPeeringConnection,
				Tags: []types.Tag{
					ec2Tag("Name", fmt.Sprintf("%v-%v", ac.infraID, target.infraID)),
				},
			},
		},
	}

	output, err := ac.client.CreateVpcPeeringConnection(context.TODO(), input)
	if err != nil {
		reporter.Failed(err)
		return nil, errors.Wrapf(err, "unable to request VPC peering")
	}

	peering := output.VpcPeeringConnection

	reporter.Succeeded("Requested VPC Peering with ID %v", peering.VpcPeeringConnectionId)

	return peering, nil
}

func (ac *awsCloud) acceptPeering(peeringID *string, reporter api.Reporter) error {
	reporter.Started("Accepting VPC Peering")

	input := &ec2.AcceptVpcPeeringConnectionInput{
		VpcPeeringConnectionId: peeringID,
	}

	_, err := ac.client.AcceptVpcPeeringConnection(context.TODO(), input)
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to accept VPC peering connection %v", peeringID)
	}

	reporter.Succeeded("Accepted VPC Peering with id: %v", peeringID)

	return nil
}

func (ac *awsCloud) createRoutesForPeering(target *awsCloud, srcVpcID, targetVpcID string,
	peering *types.VpcPeeringConnection, reporter api.Reporter) error {
	reporter.Started("Create VPC Peering")

	routeTableID, err := ac.getRouteTableID(srcVpcID, reporter)
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to create route for %v", srcVpcID)
	}

	input := &ec2.CreateRouteInput{
		RouteTableId:           routeTableID,
		DestinationCidrBlock:   peering.AccepterVpcInfo.CidrBlock,
		VpcPeeringConnectionId: peering.VpcPeeringConnectionId,
	}

	_, err = ac.client.CreateRoute(context.TODO(), input)
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to create route for %v", srcVpcID)
	}

	routeTableID, err = target.getRouteTableID(targetVpcID, reporter)
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to create route for %v", targetVpcID)
	}

	input = &ec2.CreateRouteInput{
		RouteTableId:           routeTableID,
		DestinationCidrBlock:   peering.RequesterVpcInfo.CidrBlock,
		VpcPeeringConnectionId: peering.VpcPeeringConnectionId,
	}

	_, err = target.client.CreateRoute(context.TODO(), input)
	if err != nil {
		reporter.Failed(err)
		return errors.Wrapf(err, "unable to create route for %v", targetVpcID)
	}

	reporter.Succeeded("Created Routes for VPC Peering connection %v", peering.VpcPeeringConnectionId)

	return nil
}

func (ac *awsCloud) getRouteTableID(vpcID string, reporter api.Reporter) (*string, error) {
	reporter.Started("Getting RouteTableID")

	vpcIDKeyName := "vpc-id"
	associationKey := "association.main"
	input := &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name: &vpcIDKeyName,
				Values: []string{
					vpcID,
				},
			},
			{
				Name: &associationKey,
				Values: []string{
					"true",
				},
			},
		},
	}

	output, err := ac.client.DescribeRouteTables(context.TODO(), input)
	if err != nil {
		reporter.Failed(err)
		return nil, errors.Wrapf(err, "unable to get route tables for %s", vpcID)
	}

	routeTableID := output.RouteTables[0].RouteTableId

	reporter.Succeeded("Retrieved RouteTableID %v", routeTableID)

	return routeTableID, nil
}

func runWithRetries(f func() error) error {
	var err error
	for retries := attempts; retries > 0; {
		err = f()
		if err != nil {
			retries--
			time.Sleep(waitTime * time.Second)
		} else {
			return nil
		}
	}
	return err
}
