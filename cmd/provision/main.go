package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func terminateCluster(svc *ec2.EC2) error {
	tagKey := "tag-key"
	tagValue := "k8s-role"
	statusKey := "instance-state-code"
	statusValue := "16"
	describeResult, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: &tagKey, Values: []*string{&tagValue}},
			&ec2.Filter{Name: &statusKey, Values: []*string{&statusValue}},
		},
	})
	if err != nil {
		return err
	}

	terminateIDs := []*string{}
	for i := 0; i < len(describeResult.Reservations); i++ {
		// what is a reservation?
		res := describeResult.Reservations[i]
		for j := 0; j < len(res.Instances); j++ {
			instanceID := res.Instances[j].InstanceId
			terminateIDs = append(terminateIDs, instanceID)
		}
	}

	if len(terminateIDs) > 0 {
		_, err = svc.TerminateInstances(&ec2.TerminateInstancesInput{
			InstanceIds: terminateIDs,
		})
	}

	return err
}

func masterUp(svc *ec2.EC2) (*string, error) {
	subnetID := os.Getenv("KUBETASTIC_SUBNET_ID")
	masterSecurityGroup := os.Getenv("KUBETASTIC_MASTER_SECURITY_GROUP")
	keyName := os.Getenv("KUBETASTIC_KEYPAIR")
	tagKey := "k8s-role"
	tagValue := "master"
	tagResourceType := "instance"
	userData := masterData()

	masterResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String("ami-0a86d340ea7fde077"),
		InstanceType:     aws.String("t3.small"),
		KeyName:          &keyName,
		MinCount:         aws.Int64(1),
		MaxCount:         aws.Int64(1),
		SecurityGroupIds: []*string{&masterSecurityGroup},
		SubnetId:         &subnetID,
		TagSpecifications: []*ec2.TagSpecification{&ec2.TagSpecification{
			ResourceType: &tagResourceType,
			Tags:         []*ec2.Tag{&ec2.Tag{Key: &tagKey, Value: &tagValue}}}},
		UserData: &userData,
	})
	if err != nil {
		return nil, fmt.Errorf("master failed: %s", err)
	}

	return masterResult.Instances[0].InstanceId, nil
}

func nodeUp(svc *ec2.EC2, nodeCount int) ([]*string, error) {
	subnetID := os.Getenv("KUBETASTIC_SUBNET_ID")
	nodeSecurityGroup := os.Getenv("KUBETASTIC_NODE_SECURITY_GROUP")
	keyName := os.Getenv("KUBETASTIC_KEYPAIR")
	tagKey := "k8s-role"
	tagValue := "node"
	tagResourceType := "instance"
	userData := nodeData()

	nodeResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String("ami-0a86d340ea7fde077"),
		InstanceType:     aws.String("t3.small"),
		KeyName:          &keyName,
		MinCount:         aws.Int64(int64(nodeCount)),
		MaxCount:         aws.Int64(int64(nodeCount)),
		SecurityGroupIds: []*string{&nodeSecurityGroup},
		SubnetId:         &subnetID,
		TagSpecifications: []*ec2.TagSpecification{&ec2.TagSpecification{
			ResourceType: &tagResourceType,
			Tags:         []*ec2.Tag{&ec2.Tag{Key: &tagKey, Value: &tagValue}}}},
		UserData: &userData,
	})
	if err != nil {
		return nil, err
	}

	nodeIDs := make([]*string, nodeCount, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodeIDs[i] = nodeResult.Instances[i].InstanceId
	}

	return nodeIDs, nil
}

func clusterUp(sess *session.Session, nodeCount int) error {
	svc := ec2.New(sess)

	err := terminateCluster(svc)
	if err != nil {
		return fmt.Errorf("cluster termination failed: %s", err)
	}

	_, err = masterUp(svc)
	if err != nil {
		return fmt.Errorf("master creation failed: %s", err)
	}
	_, err = nodeUp(svc, nodeCount)
	if err != nil {
		return fmt.Errorf("node creation failed: %s", err)
	}

	return nil
}

// Provision kube cluster in ec2
func main() {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	err = clusterUp(sess, 1)
	if err != nil {
		log.Fatal(err)
	}
}
