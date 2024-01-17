package nameserver

import (
	"context"
	"fmt"
	"log"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/n6g7/bingo/internal/config"
)

type Route53NS struct {
	hostedZone *string
	recordType types.RRType
	ttl        *int64
	region     string

	hostedZoneId *string
	client       *route53.Client
}

func NewRoute53NS(conf config.Route53Conf) *Route53NS {
	return &Route53NS{
		hostedZone: &conf.HostedZone,
		recordType: types.RRTypeCname,
		ttl:        &conf.TTL,
		region:     conf.AWSRegion,
	}
}

func (r *Route53NS) Init() error {
	cfg, err := awsConfig.LoadDefaultConfig(
		context.TODO(),
		awsConfig.WithRegion(r.region),
	)
	if err != nil {
		return fmt.Errorf("Error loading AWS config :%w", err)
	}

	client := route53.NewFromConfig(cfg)
	r.client = client

	// Check hosted zone exists
	output, err := r.client.ListHostedZonesByName(context.TODO(), &route53.ListHostedZonesByNameInput{
		DNSName: r.hostedZone,
	})
	if err != nil {
		return fmt.Errorf("Error listing hosted zones: %w", err)
	}
	if len(output.HostedZones) > 1 {
		return fmt.Errorf("Found multiple (%d) hosted zones matching DNS name \"%s\", try a different name?", len(output.HostedZones), *r.hostedZone)
	}
	if len(output.HostedZones) == 0 {
		return fmt.Errorf("Could not find a hosted zone with DNS name \"%s\"", *r.hostedZone)
	}
	r.hostedZoneId = output.HostedZones[0].Id
	log.Printf("[DEBUG] Found hosted zone \"%s\" with id \"%s\"", *r.hostedZone, *r.hostedZoneId)

	return nil
}

func (r *Route53NS) listRecordSets() ([]types.ResourceRecordSet, error) {
	outputs, err := r.client.ListResourceRecordSets(
		context.TODO(),
		&route53.ListResourceRecordSetsInput{
			HostedZoneId: r.hostedZoneId,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("Error listing records sets in %s: %w", *r.hostedZone, err)
	}
	return outputs.ResourceRecordSets, nil
}

func (r *Route53NS) ListRecords() (records []Record, err error) {
	rrsets, err := r.listRecordSets()
	if err != nil {
		return nil, err
	}

	for _, rrs := range rrsets {
		if rrs.Type != r.recordType {
			continue
		}
		for _, rr := range rrs.ResourceRecords {
			records = append(records, Record{
				Name:  (*rrs.Name)[:len(*rrs.Name)-1],
				Cname: *rr.Value,
			})
		}
	}
	return
}

func (r *Route53NS) RemoveRecord(name string) error {
	rrsets, err := r.listRecordSets()
	if err != nil {
		return err
	}

	var selectedRRS *types.ResourceRecordSet = nil
	for _, rrs := range rrsets {
		if *rrs.Name == name || *rrs.Name == name+"." {
			selectedRRS = &rrs
			break
		}
	}

	if selectedRRS == nil {
		return fmt.Errorf("Could not find record set for \"%s\", nothing to delete.", name)
	}

	_, err = r.client.ChangeResourceRecordSets(context.TODO(), &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: r.hostedZoneId,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action:            types.ChangeActionDelete,
					ResourceRecordSet: selectedRRS,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Error while deleting record set \"%s\": %w", name, err)
	}
	return nil
}

func (r *Route53NS) AddRecord(name, cname string) error {
	_, err := r.client.ChangeResourceRecordSets(context.TODO(), &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: r.hostedZoneId,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeActionCreate,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: &name,
						Type: r.recordType,
						TTL:  r.ttl,
						ResourceRecords: []types.ResourceRecord{
							{
								Value: &cname,
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Error while creating record \"%s\": %w", name, err)
	}
	return nil
}
