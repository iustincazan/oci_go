package main

// https://pkg.go.dev/github.com/oracle/oci-go-sdk#section-readme

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
)

var myComp = "ocid1.compartment.oc1..aaaaaaaaujiqzjnhniio5afjscxis2fx34nm3qjspnpe542wvjacosmvrdaq"
var myVCN = "ocid1.vcn.oc1.eu-frankfurt-1.amaaaaaarfznpjyav6lvnkp2232ys4eopaxizqqepric2suwhsfmnw2buaha"
var mySubnet = "ocid1.subnet.oc1.eu-frankfurt-1.aaaaaaaa3kik56pua4fzrz4cnebffc5a72ba7vnrhji6bgv6rt7lwy4dfj4q"
var SSH_NSG = "ocid1.networksecuritygroup.oc1.eu-frankfurt-1.aaaaaaaasnlrrc5rtirvauq2nq32a6rfmyhnl3bhjfme6gg5nsfmb4yxsdfq"
var instance_1 = "ocid1.instance.oc1.eu-frankfurt-1.antheljsrfznpjycuilmf3b2n4uc4txjfaxfmog6vztyhhhvjsgghl2z4llq"
var i1VNIC = "ocid1.vnic.oc1.eu-frankfurt-1.abtheljs6vfniufkspq7tcd7tbl2jvawly56njlbtk53ht4cykivm74v3d5q"
var instanceIP = "130.61.231.169"
var privateIP = "192.168.1.132"

func main() {

	// retryLauncInstance()
	// listNSGVNICs(&SSH_NSG)
	// getVNICIP(&i1VNIC)

	addNSGRule(&SSH_NSG)
}

func retryLauncInstance() {
	key, err := getSSHkey()
	if err != nil {
		fmt.Println(err.Error())
	}
	instance := core.Instance{}
	t := time.Duration(30)

	for {
		temp, err := launchInstances(&myComp, &myVCN, &mySubnet, []string{SSH_NSG}, key)
		if err != nil {
			if strings.Contains(err.Error(), "Out of host capacity") {
				time.Sleep(t)
				continue
			} else {
				if strings.Contains(err.Error(), "Too many requests for the user") {
					time.Sleep(600)
				}
				fmt.Println(err.Error())
				return
			}
		}
		instance = temp
		break
	}
	fmt.Println(instance)
}

func launchInstances(compId *string, vcnId *string, subnetId *string, nsgIds []string, ssh_key string) (core.Instance, error) {

	// launching an instance
	// https://pkg.go.dev/github.com/oracle/oci-go-sdk@v24.3.0+incompatible/core#LaunchInstanceRequest
	computeClient, err := core.NewComputeClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return core.Instance{}, err
	}

	image_id := "ocid1.image.oc1.eu-frankfurt-1.aaaaaaaavlxd4iqdgkpd3ogrnip5rdasxe3wxr5uwl65piukjlg5pvqc7peq"

	metadata := make(map[string]string)
	metadata["ssh_authorized_keys"] = ssh_key

	launchInstanceShapeConfigDetails := core.LaunchInstanceShapeConfigDetails{
		Ocpus: common.Float32(1),
	}

	createVNICDetails := core.CreateVnicDetails{
		AssignPublicIp: common.Bool(true),
		SubnetId:       common.String(*subnetId),
		NsgIds:         nsgIds,
	}

	instanceSourceViaImageDetails := core.InstanceSourceViaImageDetails{
		ImageId: common.String(image_id),
	}

	instance_details := core.LaunchInstanceDetails{
		AvailabilityDomain: common.String("tMJk:EU-FRANKFURT-1-AD-1"),
		CompartmentId:      common.String(*compId),
		Shape:              common.String("VM.Standard.A1.Flex"),
		CreateVnicDetails:  &createVNICDetails,
		ShapeConfig:        &launchInstanceShapeConfigDetails,
		SourceDetails:      instanceSourceViaImageDetails,
		Metadata:           metadata,
	}

	instance_request := core.LaunchInstanceRequest{
		LaunchInstanceDetails: instance_details,
	}
	ctx := context.Background()

	resp, err := computeClient.LaunchInstance(ctx, instance_request)
	if err != nil {
		return core.Instance{}, err
	}
	if resp.HTTPResponse().StatusCode != 200 {
		return core.Instance{}, errors.New(resp.HTTPResponse().Status)
	}
	return resp.Instance, nil
}

func listSubnets(compId *string, vcnId *string) ([]core.Subnet, error) {

	client, err := core.NewVirtualNetworkClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return nil, err
	}

	request := core.ListSubnetsRequest{
		CompartmentId: common.String(*compId),
		VcnId:         common.String(*vcnId),
	}

	ctx := context.Background()
	resp, err := client.ListSubnets(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

func addNSGRule(nsgid *string) error {

	client, err := core.NewVirtualNetworkClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return nil
	}

	rules := []core.AddSecurityRuleDetails{}

	portRange := core.PortRange{
		Min: common.Int(3300),
		Max: common.Int(3400),
	}
	tcpOptions := core.TcpOptions{
		DestinationPortRange: &portRange,
	}

	addSecurityRuleDetailsDirectionEnum := core.AddSecurityRuleDetailsDirectionEnum("INGRESS")
	addSecurityRuleDetailsSourceTypeEnum := core.AddSecurityRuleDetailsSourceTypeEnum("CIDR_BLOCK")

	ruleOne := core.AddSecurityRuleDetails{
		Direction:   addSecurityRuleDetailsDirectionEnum,
		Protocol:    common.String("6"),
		IsStateless: common.Bool(false),
		Source:      common.String("0.0.0.0/0"),
		SourceType:  addSecurityRuleDetailsSourceTypeEnum,
		TcpOptions:  &tcpOptions,
	}
	rules = append(rules, ruleOne)

	addNetworkSecurityGroupSecurityRulesDetails := core.AddNetworkSecurityGroupSecurityRulesDetails{
		SecurityRules: rules,
	}

	addNetworkSecurityGroupSecurityRulesRequest := core.AddNetworkSecurityGroupSecurityRulesRequest{
		NetworkSecurityGroupId:                      common.String(*nsgid),
		AddNetworkSecurityGroupSecurityRulesDetails: addNetworkSecurityGroupSecurityRulesDetails,
	}

	ctx := context.Background()
	resp, err := client.AddNetworkSecurityGroupSecurityRules(ctx, addNetworkSecurityGroupSecurityRulesRequest)
	if err != nil {
		return nil
	}

	if resp.HTTPResponse().StatusCode != 200 {
		return errors.New(resp.HTTPResponse().Status)
	}
	return nil
}

func listNSGs(compId *string, vcnId *string) ([]core.NetworkSecurityGroup, error) {

	client, err := core.NewVirtualNetworkClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return nil, err
	}

	request := core.ListNetworkSecurityGroupsRequest{
		CompartmentId: common.String(*compId),
		VcnId:         common.String(*vcnId),
	}

	ctx := context.Background()
	resp, err := client.ListNetworkSecurityGroups(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

func listNSGVNICs(nsgId *string) {
	client, err := core.NewVirtualNetworkClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	listNetworkSecurityGroupVnicsRequest := core.ListNetworkSecurityGroupVnicsRequest{
		NetworkSecurityGroupId: common.String(SSH_NSG),
	}
	ctx := context.Background()
	resp, err := client.ListNetworkSecurityGroupVnics(ctx, listNetworkSecurityGroupVnicsRequest)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(resp.Items)
}

func getVNICIP(vnic *string) {
	client, err := core.NewVirtualNetworkClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	getVnicRequest := core.GetVnicRequest{
		VnicId: common.String(*vnic),
	}
	ctx := context.Background()
	resp, err := client.GetVnic(ctx, getVnicRequest)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(*resp.PublicIp)
}

func getSSHkey() (string, error) {
	keyBytes, err := ioutil.ReadFile("/Users/iustin/.ssh/id_rsa.pub")
	if err != nil {
		return "", err
	}
	return string(keyBytes), nil
}

// Examples
// https://github.com/oracle/oci-go-sdk/tree/master/example
