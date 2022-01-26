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
	//Context("Test Create AWS Peering", testCreateAWSPeering)
	Context("Test Get Route Table ID", testGetRouteTableID)
	Context("Test Request Peering", testRequestPeering)
	Context("Test Accept Peering", testAcceptPeering)
	Context("Test Create Routes for Peering", testCreateRoutesForPeering)
})

func testRequestPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	var _ = Describe("Validate error input", func() {
		When("Create a Peering Request", func() {
			It("works", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				awsCloudB := cloudB.cloud.(*awsCloud)

				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "10.0.0.0/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-b", "10.1.0.0/16"))

				vpcA, _ := awsCloudA.getVpcID()
				vpcB, _ := awsCloudB.getVpcID()

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
			It("not works", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				awsCloudB := cloudB.cloud.(*awsCloud)

				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "10.0.0.0/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-b", "10.1.0.0/16"))

				vpcA, _ := awsCloudA.getVpcID()
				vpcB, _ := awsCloudB.getVpcID()
				errMsg := "unable to request VPC peering"

				// TODO: Should we add more fields to mocked response?
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

	var _ = Describe("Validate error input", func() {
		When("Accept a Peering Request", func() {
			It("works", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				peeringID := "peer-id"

				cloudA.awsClient.EXPECT().AcceptVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, nil)

				err := awsCloudA.acceptPeering(&peeringID, api.NewLoggingReporter())
				Expect(err).To(BeNil())
			})
			It("not works", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				peeringID := "peer-id"

				cloudA.awsClient.EXPECT().AcceptVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Accept Peering Error"))

				err := awsCloudA.acceptPeering(&peeringID, api.NewLoggingReporter())
				Expect(err).NotTo(BeNil())
			})
		})
	})
}

// TODO
func testCreateRoutesForPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	vpcAID := "vpc-a"
	vpcBID := "vpc-b"
	peeringID := "peer-id"
	peering := types.VpcPeeringConnection{
		AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
			VpcId: &vpcAID,
		},
		RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
			VpcId: &vpcBID,
		},
		VpcPeeringConnectionId: &peeringID,
	}

	var _ = Describe("Validate error input", func() {
		When("Create Routes For peering", func() {
			It("works", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				awsCloudB := cloudB.cloud.(*awsCloud)
				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcAID, "10.0.0.0/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcBID, "10.1.0.0/16"))
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcAID))
				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcBID))
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)

				vpcA, _ := awsCloudA.getVpcID()
				vpcB, _ := awsCloudB.getVpcID()

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())
				Expect(err).To(BeNil())
			})
			It("Can't create route on requester", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				awsCloudB := cloudB.cloud.(*awsCloud)

				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcAID, "10.0.0.0/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcAID, "10.1.0.0/16"))
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcAID))
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Can't create route"))

				vpcA, _ := awsCloudA.getVpcID()
				vpcB, _ := awsCloudB.getVpcID()

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())
				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcA)))
			})
			It("Can't create route on accepter", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				awsCloudB := cloudB.cloud.(*awsCloud)
				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcAID, "10.0.0.0/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcBID, "10.1.0.0/16"))
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcAID))
				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcBID))
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Can't create route"))

				vpcA, _ := awsCloudA.getVpcID()
				vpcB, _ := awsCloudB.getVpcID()

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())
				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcB)))
			})
			It("Can't get Requester Route Table", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				awsCloudB := cloudB.cloud.(*awsCloud)

				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcAID, "10.0.0.0/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcBID, "10.1.0.0/16"))

				vpcA, _ := awsCloudA.getVpcID()
				vpcB, _ := awsCloudB.getVpcID()

				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to create route for "+vpcA))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())
				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcA)))
			})
			It("Can't get Accepter Route Table", func() {
				awsCloudA := cloudA.cloud.(*awsCloud)
				awsCloudB := cloudB.cloud.(*awsCloud)

				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcAID, "10.0.0.0/16"))
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor(vpcBID, "10.1.0.0/16"))

				vpcA, _ := awsCloudA.getVpcID()
				vpcB, _ := awsCloudB.getVpcID()

				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcAID))
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
	cloud := newCloudTestDriver(infraID, region)
	var _ = Describe("Test Get Route Table ID", func() {
		When("Trying to get Route Table ID", func() {
			It("returns correct Route Table ID", func() {
				vpcID := "vpc-a"
				cloud.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcID))

				awsCloud := cloud.cloud.(*awsCloud)
				rtID, err := awsCloud.getRouteTableId(vpcID, api.NewLoggingReporter())
				Expect(err).To(BeNil())
				Expect(rtID).ToNot(BeNil())
				Expect(rtID).ToNot(Equal(vpcID + "-rt"))
			})
		})
		It("returns an error", func() {
			errMsg := "Route Table not Found"
			cloud.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
				Return(nil, errors.New(errMsg))

			awsCloud := cloud.cloud.(*awsCloud)
			rtID, err := awsCloud.getRouteTableId("", api.NewLoggingReporter())
			Expect(err).ToNot(BeNil())
			Expect(err).Should(
				MatchError(
					MatchRegexp(errMsg),
				),
			)
			Expect(rtID).To(BeNil())
		})
	})
}
