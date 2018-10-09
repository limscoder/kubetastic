package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const instanceStatusRunning = int64(16)

func terminateCluster(svc *ec2.EC2) error {
	tagKey := "tag-key"
	tagValue := "k8s-role"
	statusKey := "instance-state-code"
	statusValue := fmt.Sprintf("%i", instanceStatusRunning)
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

func masterUp(svc *ec2.EC2) (*ec2.Instance, error) {
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

	return waitForMasterInit(svc, masterResult.Instances[0])
}

func waitForMasterInit(svc *ec2.EC2, master *ec2.Instance) (*ec2.Instance, error) {
	// Wait for vm statusResult
	for {
		fmt.Println("waiting for VM ready")
		statusResult, err := svc.DescribeInstanceStatus(
			&ec2.DescribeInstanceStatusInput{
				InstanceIds: []*string{master.InstanceId},
			})
		if err != nil {
			return nil, err
		}

		isReady := len(statusResult.InstanceStatuses) == 1 && *statusResult.InstanceStatuses[0].InstanceState.Code == instanceStatusRunning
		if isReady {
			// re-query to get public ip
			describeResult, err := svc.DescribeInstances(
				&ec2.DescribeInstancesInput{
					InstanceIds: []*string{master.InstanceId},
				})
			if err != nil {
				return nil, err
			}
			master = describeResult.Reservations[0].Instances[0]

			// wait for kube master node ready
			args := []string{
				"-o", "StrictHostKeyChecking no",
				"-i",
				fmt.Sprintf("~/.ssh/%s.pem", os.Getenv("KUBETASTIC_KEYPAIR")),
				fmt.Sprintf("core@%s", *master.PublicIpAddress),
				"stat $HOME/.kube/READY"}
			for {
				fmt.Println("waiting for master ready", *master.PublicIpAddress)
				cmd := exec.Command("ssh", args...)
				if _, err := cmd.CombinedOutput(); err == nil {
					return master, nil
				}
				time.Sleep(10 * time.Second)
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func joinCmd(master *ec2.Instance) (string, error) {
	args := []string{
		"-o", "StrictHostKeyChecking no",
		"-i",
		fmt.Sprintf("~/.ssh/%s.pem", os.Getenv("KUBETASTIC_KEYPAIR")),
		fmt.Sprintf("core@%s", *master.PublicIpAddress),
		"sudo kubeadm token create --print-join-command"}
	cmd := exec.Command("ssh", args...)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func nodeUp(svc *ec2.EC2, master *ec2.Instance, nodeCount int) ([]*ec2.Instance, error) {
	subnetID := os.Getenv("KUBETASTIC_SUBNET_ID")
	nodeSecurityGroup := os.Getenv("KUBETASTIC_NODE_SECURITY_GROUP")
	keyName := os.Getenv("KUBETASTIC_KEYPAIR")
	tagKey := "k8s-role"
	tagValue := "node"
	tagResourceType := "instance"
	join, err := joinCmd(master)
	if err != nil {
		return nil, err
	}
	userData := nodeData(join)

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

	nodes := make([]*ec2.Instance, nodeCount, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes[i] = nodeResult.Instances[i]
	}

	return nodes, nil
}

func clusterUp(sess *session.Session, nodeCount int) error {
	svc := ec2.New(sess)

	err := terminateCluster(svc)
	if err != nil {
		return fmt.Errorf("cluster termination failed: %s", err)
	}

	master, err := masterUp(svc)
	if err != nil {
		return fmt.Errorf("master creation failed: %s", err)
	}
	fmt.Println("master initialized")

	_, err = nodeUp(svc, master, nodeCount)
	if err != nil {
		return fmt.Errorf("node creation failed: %s", err)
	}
	fmt.Println("nodes initializing...")

	return nil
}

// super janky kops knockoff
// Provisions kube cluster in ec2
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
