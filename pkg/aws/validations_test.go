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
	"github.com/aws/smithy-go"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/cloud-prepare/pkg/api"
)

var _ = Describe("AWS Peering", func() {
	Context("Test Validate Peering Prerequisites", testValidatePeeringPrerequisites)
	Context("Test Check VPC Overlap", testCheckVpcOverlap)
	Context("Test Determine Permission Error", testDeterminePermissionError)
})

func testDeterminePermissionError() {
	var _ = Describe("Validate error input", func() {
		When("Args are wrong", func() {
			It("err is nil", func() {
				result := determinePermissionError(nil, "")
				Expect(result).To(BeNil())
			})
		})
		When("It's an AWS error", func() {
			It("is DryRunOperation error", func() {
				err := smithy.GenericAPIError{
					Code:    "DryRunOperation",
					Message: "DryRunOperation",
					Fault:   1,
				}
				operation := "check"
				result := determinePermissionError(&err, operation)
				Expect(result).Should(BeNil())
			})
			It("is UnauthorizedOperation error", func() {
				err := smithy.GenericAPIError{
					Code:    "UnauthorizedOperation",
					Message: "UnauthorizedOperation",
					Fault:   1,
				}
				operation := "check"
				result := determinePermissionError(&err, operation)
				Expect(result).Should(
					MatchError(
						MatchRegexp("no permission to " + operation),
					),
				)
			})
		})
		When("It's not an AWS error", func() {
			It("is a Generic Error", func() {
				err := smithy.GenericAPIError{
					Code:    "Generic Error",
					Message: "Generic Error",
					Fault:   1,
				}
				operation := ""
				result := determinePermissionError(&err, operation)
				Expect(result).Should(
					MatchError(
						MatchRegexp("error while checking permissions for " + operation),
					),
				)
			})
		})
	})
}

func testValidatePeeringPrerequisites() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	var _ = Describe("Validate Validate Peering Prerequisites", func() {
		When("trying to retrieve the VPC", func() {
			It("cannot retrieve the source VPC", func() {
				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(
						&ec2.DescribeVpcsOutput{
							Vpcs: []types.Vpc{},
						}, nil)
				awsCloudA, ok := cloudA.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				awsCloudB, ok := cloudB.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				err := awsCloudA.validatePeeringPrerequisites(awsCloudB, api.NewLoggingReporter())

				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError(And(
					MatchRegexp("unable to validate vpc peering prerequisites for source"),
					MatchRegexp("not found VPC test-infraID-vpc"),
				)))
			})
			It("cannot retrieve the target VPC", func() {
				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "1.2.3.4/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(
						&ec2.DescribeVpcsOutput{
							Vpcs: []types.Vpc{},
						}, nil)
				awsCloudA, ok := cloudA.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				awsCloudB, ok := cloudB.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				err := awsCloudA.validatePeeringPrerequisites(awsCloudB, api.NewLoggingReporter())

				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError(And(
					MatchRegexp("unable to validate vpc peering prerequisites for target"),
					MatchRegexp("not found VPC other-infraID-vpc"),
				)))
			})
		})
		When("checking if VPCs overlap", func() {
			It("fails with an invalid CIDR Block", func() {
				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "make it fail"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-b", "1.2.3.4/16"))
				awsCloudA, ok := cloudA.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				awsCloudB, ok := cloudB.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				err := awsCloudA.validatePeeringPrerequisites(awsCloudB, api.NewLoggingReporter())

				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError("invalid CIDR address: make it fail"))
			})
			It("fails with overlapping CIDR BLocks", func() {
				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "1.2.3.4/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-b", "1.2.3.4/16"))
				awsCloudA, ok := cloudA.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				awsCloudB, ok := cloudB.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				err := awsCloudA.validatePeeringPrerequisites(awsCloudB, api.NewLoggingReporter())

				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError("source [1.2.3.4/16] and target [1.2.3.4/16] CIDR Blocks must not overlap"))
			})
		})
		When("requirements are met", func() {
			It("returns with no error", func() {
				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "10.0.0.0/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-b", "10.1.0.0/16"))
				awsCloudA, ok := cloudA.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				awsCloudB, ok := cloudB.cloud.(*awsCloud)
				Expect(ok).To(BeTrue())
				err := awsCloudA.validatePeeringPrerequisites(awsCloudB, api.NewLoggingReporter())

				Expect(err).ShouldNot(HaveOccurred())
			})
		})
	})
}

func testCheckVpcOverlap() {
	var _ = Describe("Validate CIDR Blocks", func() {
		When("Invalid CIDR block", func() {
			var (
				vpcA, vpcB *types.Vpc
			)
			It("netA is invalid", func() {
				netA := "1.2.3.4/-1"
				netB := "10.0.0.0/16"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, err := CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeFalse())
				Expect(err).NotTo(BeNil())
			})
			It("netB is invalid", func() {
				netA := "10.0.0.0/16"
				netB := "1.2.3.4/-1"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, err := CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeFalse())
				Expect(err).NotTo(BeNil())
			})
		})
		When("CIDR blocks not overlap", func() {
			var (
				vpcA, vpcB *types.Vpc
			)
			It("Same mask different subnet", func() {
				netA := "10.0.0.0/16"
				netB := "10.1.0.0/16"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, _ := CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeFalse())
			})
		})
		When("CIDR blocks overlap", func() {
			var (
				vpcA, vpcB *types.Vpc
			)
			It("Same CIDR", func() {
				netA := "10.0.0.0/16"
				netB := "10.0.0.0/16"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, _ := CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeTrue())
			})
			It("Same mask different subnet", func() {
				netA := "10.0.0.0/12"
				netB := "10.1.0.0/12"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, _ := CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeTrue())
			})
			It("Specify IP in all CIDR block", func() {
				netA := "192.168.0.1/32"
				netB := "0.0.0.0/0"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, _ := CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeTrue())
			})
		})
	})
}
