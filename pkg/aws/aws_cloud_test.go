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

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/submariner-io/cloud-prepare/pkg/api"
)

var _ = Describe("AWS Peering", func() {
	Context("Accept Peering", testCreateVpcPeering)
})

func testCreateVpcPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	var _ = Describe("VPC Peering", func() {
		When("receiving a target Cloud", func() {
			It("is an unsupported Cloud", func() {
				invalidCloud := &fooCloud{}
				err := cloudA.cloud.CreateVpcPeering(invalidCloud, api.NewLoggingReporter())
				Expect(err).Should(MatchError("only AWS clients are supported"))
			})
		})
		When("prerequisites are not met", func() {
			It("receives an overlapping CidrBlock for source and target", func() {
				cloudA.awsClient.EXPECT().
					DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "10.0.0.0/12"))

				cloudB.awsClient.EXPECT().
					DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-b", "10.1.0.0/12"))
				err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())
				Expect(err).To(HaveOccurred())
				Expect(err).Should(
					MatchError("unable to validate vpc peering prerequisites: source [10.0.0.0/12] and target [10.1.0.0/12] CIDR Blocks must not overlap"),
				)
			})
		})
		When("retrieving the VPC IDs", func() {
			It("fails to get the source VPC ID", func() {
				ensurePrerequisitesAreMet(cloudA, cloudB)
				cloudA.awsClient.EXPECT().DescribeVpcs(gomock.Any(), gomock.Any()).
					Return(nil, errors.Errorf("some error"))
				err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).Should(MatchError(MatchRegexp("unable to retrieve source VPC ID")))
			})
			It("fails to get the target VPC ID", func() {
				ensurePrerequisitesAreMet(cloudA, cloudB)
				cloudA.awsClient.EXPECT().
					DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "10.0.0.0/12"))
				cloudB.awsClient.EXPECT().DescribeVpcs(gomock.Any(), gomock.Any()).
					Return(nil, errors.Errorf("some error"))
				err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).Should(MatchError(MatchRegexp("unable to retrieve target VPC ID")))
			})
		})
	})
}

type fooCloud struct {
}

func (f *fooCloud) PrepareForSubmariner(input api.PrepareForSubmarinerInput, reporter api.Reporter) error {
	panic("not implemented")
}

func (f *fooCloud) CreateVpcPeering(target api.Cloud, reporter api.Reporter) error {
	panic("not implemented")
}

func (f *fooCloud) CleanupAfterSubmariner(reporter api.Reporter) error {
	panic("not implemented")
}

func getRouteTableFor(vpcID string) (*ec2.DescribeRouteTablesOutput, error) {
	rtID := vpcID + "-rt"
	return &ec2.DescribeRouteTablesOutput{
		RouteTables: []types.RouteTable{
			{
				VpcId:        &vpcID,
				RouteTableId: &rtID,
			},
		},
	}, nil
}

func getVpcOutputFor(id, cidrBlock string) (*ec2.DescribeVpcsOutput, error) {
	return &ec2.DescribeVpcsOutput{
		Vpcs: []types.Vpc{
			{
				VpcId:     &id,
				CidrBlock: &cidrBlock,
			},
		},
	}, nil
}

type cloudTestDriver struct {
	fakeAWSClientBase
	cloud api.Cloud
}

func newCloudTestDriver(infraID, region string) *cloudTestDriver {
	t := &cloudTestDriver{}

	BeforeEach(func() {
		t.beforeEach()
		t.cloud = NewCloud(t.awsClient, infraID, region)
	})

	AfterEach(t.afterEach)

	return t
}

func ensurePrerequisitesAreMet(cloudA, cloudB *cloudTestDriver) {
	cloudA.awsClient.EXPECT().
		DescribeVpcs(context.TODO(), gomock.Any()).
		Return(getVpcOutputFor("vpc-a", "10.0.0.0/16")).
		Times(1)

	cloudB.awsClient.EXPECT().
		DescribeVpcs(context.TODO(), gomock.Any()).
		Return(getVpcOutputFor("vpc-b", "10.1.0.0/16")).
		Times(1)
}
