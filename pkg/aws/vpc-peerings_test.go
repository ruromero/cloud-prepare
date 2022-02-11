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
	Context("Test Get Route Table ID", testGetRouteTableID)
	Context("Test Request Peering", testRequestPeering)
	Context("Test Accept Peering", testAcceptPeering)
	Context("Test Create Routes for Peering", testCreateRoutesForPeering)
})

func testRequestPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	vpcA := "vpc-a"
	vpcB := "vpc-b"
	var awsCloudA, awsCloudB *awsCloud
	var ok bool
	_ = Describe("Validate Peering request", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
			awsCloudB, ok = cloudB.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Create , a Peering Request", func() {
			It("can request it", func() {
				// TODO: Should we add more fields to mocked response?
				cloudA.awsClient.EXPECT().CreateVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(&ec2.CreateVpcPeeringConnectionOutput{
						VpcPeeringConnection: &types.VpcPeeringConnection{
							RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
								VpcId:  &vpcA,
								Region: &awsCloudA.region,
							},
						},
					}, nil)

				vpcPeering, err := awsCloudA.requestPeering(vpcA, vpcB, awsCloudB, api.NewLoggingReporter())

				Expect(err).To(BeNil())
				Expect(vpcPeering).NotTo(BeNil())
				Expect(*vpcPeering.RequesterVpcInfo.Region).To(Equal(awsCloudA.region))
				Expect(*vpcPeering.RequesterVpcInfo.VpcId).To(Equal(vpcA))
			})
			It("can't request a VPC peering", func() {
				errMsg := "unable to request VPC peering"

				cloudA.awsClient.EXPECT().CreateVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, errors.New(errMsg))

				vpcPeering, err := awsCloudA.requestPeering(vpcA, vpcB, awsCloudB, api.NewLoggingReporter())

				Expect(err).Should(MatchError(MatchRegexp(errMsg)))
				Expect(vpcPeering).To(BeNil())
			})
		})
	})
}

func testAcceptPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	peeringID := "peer-id"
	var awsCloudA *awsCloud
	var ok bool
	_ = Describe("Validate Accept peering process", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Trying to accept a Peering Request", func() {
			It("is accepted", func() {
				cloudA.awsClient.EXPECT().AcceptVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, nil)

				err := awsCloudA.acceptPeering(&peeringID, api.NewLoggingReporter())
				Expect(err).To(BeNil())
			})
			It("is not accepted", func() {
				cloudA.awsClient.EXPECT().AcceptVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Accept Peering Error"))

				err := awsCloudA.acceptPeering(&peeringID, api.NewLoggingReporter())
				Expect(err).NotTo(BeNil())
			})
		})
	})
}

func testCreateRoutesForPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	vpcA := "vpc-a"
	vpcB := "vpc-b"
	peeringID := "peer-id"
	peering := types.VpcPeeringConnection{
		AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
			VpcId: &vpcA,
		},
		RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
			VpcId: &vpcB,
		},
		VpcPeeringConnectionId: &peeringID,
	}
	var awsCloudA, awsCloudB *awsCloud
	var ok bool
	_ = Describe("Validate error input", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
			awsCloudB, ok = cloudB.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Create Routes For peering", func() {
			It("can create them", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA))
				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcB))
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).To(BeNil())
			})
			It("Can't create route on requester", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA))
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Can't create route"))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcA)))
			})
			It("Can't create route on accepter", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA))
				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcB))
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Can't create route"))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcB)))
			})
			It("Can't get Requester Route Table", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to create route for "+vpcA))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcA)))
			})
			It("Can't get Accepter Route Table", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA))
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to create route for "+vpcB))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcB)))
			})
		})
	})
}

func testGetRouteTableID() {
	cloudA := newCloudTestDriver(infraID, region)
	vpcA := "vpc-a"
	var awsCloudA *awsCloud
	var ok bool
	_ = Describe("Test Get Route Table ID", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Trying to get Route Table ID", func() {
			It("returns correct Route Table ID", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA))

				rtID, err := awsCloudA.getRouteTableID(vpcA, api.NewLoggingReporter())

				Expect(err).To(BeNil())
				Expect(rtID).ToNot(BeNil())
				Expect(rtID).ToNot(Equal(vpcA + "-rt"))
			})
		})
		It("can't return Route Table ID", func() {
			errMsg := "Route Table not Found"

			cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
				Return(nil, errors.New(errMsg))

			rtID, err := awsCloudA.getRouteTableID("", api.NewLoggingReporter())

			Expect(err).ToNot(BeNil())
			Expect(err).Should(MatchError(MatchRegexp(errMsg)))
			Expect(rtID).To(BeNil())
		})
	})
}
