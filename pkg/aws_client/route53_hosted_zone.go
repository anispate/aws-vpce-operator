/*
Copyright 2022.

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

package aws_client

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/openshift/aws-vpce-operator/pkg/util"
	"time"
)

// ListHostedZonesByVPC is a wrapper around route53:ListHostedZonesByVPC
// TODO: What does this do
func (c *AWSClient) ListHostedZonesByVPC(ctx context.Context, vpc, region string) (*route53.ListHostedZonesByVPCOutput, error) {
	input := &route53.ListHostedZonesByVPCInput{
		VPCId:     aws.String(vpc),
		VPCRegion: types.VPCRegion(region),
	}

	// TODO: Unlikely, but would be nice to handle pagination
	return c.route53Client.ListHostedZonesByVPC(ctx, input)
}

// ListResourceRecordSets returns a list of records for a given hosted zone ID
func (c *AWSClient) ListResourceRecordSets(ctx context.Context, hostedZoneId string) (*route53.ListResourceRecordSetsOutput, error) {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneId),
	}

	// TODO: Handle pagination
	resp, err := c.route53Client.ListResourceRecordSets(ctx, input)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// UpsertResourceRecordSet updates or creates a resource record set
func (c *AWSClient) UpsertResourceRecordSet(ctx context.Context, rrs *types.ResourceRecordSet, hostedZoneId string) (*route53.ChangeResourceRecordSetsOutput, error) {
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					// Upsert behavior: If a resource record set doesn't already exist, Route 53 creates it.
					// If a resource record set does exist, Route 53 updates it with the values in the request.
					Action:            types.ChangeActionUpsert,
					ResourceRecordSet: rrs,
				},
			},
		},
		HostedZoneId: aws.String(hostedZoneId),
	}

	return c.route53Client.ChangeResourceRecordSets(ctx, input)
}

// DeleteResourceRecordSet deletes a specific record from a hosted zone
// NOTE: To delete a resource record set, you must specify all the same values that you specified when you created it.
func (c *AWSClient) DeleteResourceRecordSet(ctx context.Context, rrs *types.ResourceRecordSet, hostedZoneId string) (*route53.ChangeResourceRecordSetsOutput, error) {
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action:            types.ChangeActionDelete,
					ResourceRecordSet: rrs,
				},
			},
			Comment: aws.String(fmt.Sprintf("Deleting %s", *rrs.Name)),
		},
		HostedZoneId: aws.String(hostedZoneId),
	}

	return c.route53Client.ChangeResourceRecordSets(ctx, input)
}

// CreateHostedZone creates a Route 53 Private Hosted Zone with the specified domain, associated to the specified
// vpcId + region.
func (c *AWSClient) CreateHostedZone(ctx context.Context, domain, vpcId, region string) (*route53.CreateHostedZoneOutput, error) {
	zoneInput := &route53.CreateHostedZoneInput{
		CallerReference:  aws.String(time.Now().String()),
		Name:             aws.String(domain),
		HostedZoneConfig: &types.HostedZoneConfig{PrivateZone: true},
		VPC:              &types.VPC{VPCId: aws.String(vpcId), VPCRegion: types.VPCRegion(region)},
	}
	return c.route53Client.CreateHostedZone(ctx, zoneInput)
}

// DeleteHostedZone deletes a Route 53 Hosted Zone by ID
func (c *AWSClient) DeleteHostedZone(ctx context.Context, id string) (*route53.DeleteHostedZoneOutput, error) {
	return c.route53Client.DeleteHostedZone(ctx, &route53.DeleteHostedZoneInput{Id: aws.String(id)})
}

// GenerateDefaultTagsForHostedZoneInput generates the ChangeTagsForResourceInput using the default tags for the zoneId
func (c *AWSClient) GenerateDefaultTagsForHostedZoneInput(zoneId, clusterTagKey string) (*route53.ChangeTagsForResourceInput, error) {
	defaultTags, err := util.GenerateR53Tags(clusterTagKey)

	changeTagsInput := &route53.ChangeTagsForResourceInput{
		ResourceId:    aws.String(zoneId),
		ResourceType:  types.TagResourceTypeHostedzone,
		AddTags:       defaultTags,
		RemoveTagKeys: nil,
	}
	return changeTagsInput, err
}

// FetchPrivateZoneTags takes context and a Route53 ZoneID and returns the output provided by ListTagsForResource for a hosted zone
func (c *AWSClient) FetchPrivateZoneTags(ctx context.Context, zoneId string) (*route53.ListTagsForResourceOutput, error) {
	return c.route53Client.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
		ResourceId:   aws.String(zoneId),
		ResourceType: types.TagResourceTypeHostedzone,
	})
}
