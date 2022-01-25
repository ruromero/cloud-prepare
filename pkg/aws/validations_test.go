package aws_test

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/cloud-prepare/pkg/aws"
)

var _ = Describe("AWS Peering", func() {
	Context("Test Validations", testValidations)
	Context("Test Peering Prerequisites", testPeeringPrerequisites)
})

func testValidations() {
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
				response, err := aws.CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeFalse())
				Expect(err).NotTo(BeNil())
			})
			It("netB is invalid", func() {
				netA := "10.0.0.0/16"
				netB := "1.2.3.4/-1"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, err := aws.CheckVpcOverlap(vpcA, vpcB)
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
				response, _ := aws.CheckVpcOverlap(vpcA, vpcB)
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
				response, _ := aws.CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeTrue())
			})
			It("Same mask different subnet", func() {
				netA := "10.0.0.0/12"
				netB := "10.1.0.0/12"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, _ := aws.CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeTrue())
			})
			It("Specifi IP in all CIDR block", func() {
				netA := "192.168.0.1/32"
				netB := "0.0.0.0/0"
				vpcA = &types.Vpc{CidrBlock: &netA}
				vpcB = &types.Vpc{CidrBlock: &netB}
				response, _ := aws.CheckVpcOverlap(vpcA, vpcB)
				Expect(response).To(BeTrue())
			})
		})
	})
}

func testPeeringPrerequisites() {
}
