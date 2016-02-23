package aws

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAutoScalingGroup_basic(t *testing.T) {
	var group autoscaling.Group
	var lc autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAWSAutoScalingGroupHealthyCapacity(&group, 2),
					testAccCheckAWSAutoScalingGroupAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "availability_zones.2487133097", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "name", "foobar3-terraform-test"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "max_size", "5"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "min_size", "2"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "health_check_grace_period", "300"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "health_check_type", "ELB"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "desired_capacity", "4"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "force_delete", "true"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.0", "OldestInstance"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.1", "ClosestToNextInstanceHour"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.new", &lc),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "desired_capacity", "5"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.0", "ClosestToNextInstanceHour"),
					testLaunchConfigurationName("aws_autoscaling_group.bar", &lc),
					testAccCheckAutoscalingTags(&group.Tags, "Bar", map[string]interface{}{
						"value":               "bar-foo",
						"propagate_at_launch": true,
					}),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroup_autoGeneratedName(t *testing.T) {
	asgNameRegexp := regexp.MustCompile("^tf-asg-")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig_autoGeneratedName,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"aws_autoscaling_group.bar", "name", asgNameRegexp),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroup_terminationPolicies(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig_terminationPoliciesEmpty,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.#", "0"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig_terminationPoliciesUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.0", "OldestInstance"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig_terminationPoliciesExplicitDefault,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.0", "Default"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig_terminationPoliciesEmpty,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroup_tags(t *testing.T) {
	var group autoscaling.Group

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAutoscalingTags(&group.Tags, "Foo", map[string]interface{}{
						"value":               "foo-bar",
						"propagate_at_launch": true,
					}),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAutoscalingTagNotExists(&group.Tags, "Foo"),
					testAccCheckAutoscalingTags(&group.Tags, "Bar", map[string]interface{}{
						"value":               "bar-foo",
						"propagate_at_launch": true,
					}),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroup_VpcUpdates(t *testing.T) {
	var group autoscaling.Group

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfigWithAZ,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "availability_zones.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "availability_zones.2487133097", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "vpc_zone_identifier.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfigWithVPCIdent,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAWSAutoScalingGroupAttributesVPCZoneIdentifer(&group),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "availability_zones.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "availability_zones.2487133097", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "vpc_zone_identifier.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroup_WithLoadBalancer(t *testing.T) {
	var group autoscaling.Group

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccAWSAutoScalingGroupConfigWithLoadBalancer),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAWSAutoScalingGroupAttributesLoadBalancer(&group),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroup_withPlacementGroup(t *testing.T) {
	var group autoscaling.Group

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig_withPlacementGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "placement_group", "test"),
				),
			},
		},
	})
}

func testAccCheckAWSAutoScalingGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_autoscaling_group" {
			continue
		}

		// Try to find the Group
		describeGroups, err := conn.DescribeAutoScalingGroups(
			&autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: []*string{aws.String(rs.Primary.ID)},
			})

		if err == nil {
			if len(describeGroups.AutoScalingGroups) != 0 &&
				*describeGroups.AutoScalingGroups[0].AutoScalingGroupName == rs.Primary.ID {
				return fmt.Errorf("AutoScaling Group still exists")
			}
		}

		// Verify the error
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidGroup.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSAutoScalingGroupAttributes(group *autoscaling.Group) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *group.AvailabilityZones[0] != "us-west-2a" {
			return fmt.Errorf("Bad availability_zones: %#v", group.AvailabilityZones[0])
		}

		if *group.AutoScalingGroupName != "foobar3-terraform-test" {
			return fmt.Errorf("Bad name: %s", *group.AutoScalingGroupName)
		}

		if *group.MaxSize != 5 {
			return fmt.Errorf("Bad max_size: %d", *group.MaxSize)
		}

		if *group.MinSize != 2 {
			return fmt.Errorf("Bad max_size: %d", *group.MinSize)
		}

		if *group.HealthCheckType != "ELB" {
			return fmt.Errorf("Bad health_check_type,\nexpected: %s\ngot: %s", "ELB", *group.HealthCheckType)
		}

		if *group.HealthCheckGracePeriod != 300 {
			return fmt.Errorf("Bad health_check_grace_period: %d", *group.HealthCheckGracePeriod)
		}

		if *group.DesiredCapacity != 4 {
			return fmt.Errorf("Bad desired_capacity: %d", *group.DesiredCapacity)
		}

		if *group.LaunchConfigurationName == "" {
			return fmt.Errorf("Bad launch configuration name: %s", *group.LaunchConfigurationName)
		}

		t := &autoscaling.TagDescription{
			Key:               aws.String("Foo"),
			Value:             aws.String("foo-bar"),
			PropagateAtLaunch: aws.Bool(true),
			ResourceType:      aws.String("auto-scaling-group"),
			ResourceId:        group.AutoScalingGroupName,
		}

		if !reflect.DeepEqual(group.Tags[0], t) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.Tags[0],
				t)
		}

		return nil
	}
}

func testAccCheckAWSAutoScalingGroupAttributesLoadBalancer(group *autoscaling.Group) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(group.LoadBalancerNames) != 1 {
			return fmt.Errorf("Bad load_balancers: %v", group.LoadBalancerNames)
		}

		return nil
	}
}

func testAccCheckAWSAutoScalingGroupExists(n string, group *autoscaling.Group) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No AutoScaling Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

		describeGroups, err := conn.DescribeAutoScalingGroups(
			&autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: []*string{aws.String(rs.Primary.ID)},
			})

		if err != nil {
			return err
		}

		if len(describeGroups.AutoScalingGroups) != 1 ||
			*describeGroups.AutoScalingGroups[0].AutoScalingGroupName != rs.Primary.ID {
			return fmt.Errorf("AutoScaling Group not found")
		}

		*group = *describeGroups.AutoScalingGroups[0]

		return nil
	}
}

func testLaunchConfigurationName(n string, lc *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if *lc.LaunchConfigurationName != rs.Primary.Attributes["launch_configuration"] {
			return fmt.Errorf("Launch configuration names do not match")
		}

		return nil
	}
}

func testAccCheckAWSAutoScalingGroupHealthyCapacity(
	g *autoscaling.Group, exp int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		healthy := 0
		for _, i := range g.Instances {
			if i.HealthStatus == nil {
				continue
			}
			if strings.EqualFold(*i.HealthStatus, "Healthy") {
				healthy++
			}
		}
		if healthy < exp {
			return fmt.Errorf("Expected at least %d healthy, got %d.", exp, healthy)
		}
		return nil
	}
}

func testAccCheckAWSAutoScalingGroupAttributesVPCZoneIdentifer(group *autoscaling.Group) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Grab Subnet Ids
		var subnets []string
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_subnet" {
				continue
			}
			subnets = append(subnets, rs.Primary.Attributes["id"])
		}

		if group.VPCZoneIdentifier == nil {
			return fmt.Errorf("Bad VPC Zone Identifier\nexpected: %s\ngot nil", subnets)
		}

		zones := strings.Split(*group.VPCZoneIdentifier, ",")

		remaining := len(zones)
		for _, z := range zones {
			for _, s := range subnets {
				if z == s {
					remaining--
				}
			}
		}

		if remaining != 0 {
			return fmt.Errorf("Bad VPC Zone Identifier match\nexpected: %s\ngot:%s", zones, subnets)
		}

		return nil
	}
}

const testAccAWSAutoScalingGroupConfig_autoGeneratedName = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  desired_capacity = 0
  max_size = 0
  min_size = 0
  launch_configuration = "${aws_launch_configuration.foobar.name}"
}
`

const testAccAWSAutoScalingGroupConfig_terminationPoliciesEmpty = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  max_size = 0
  min_size = 0
  desired_capacity = 0

  launch_configuration = "${aws_launch_configuration.foobar.name}"
}
`

const testAccAWSAutoScalingGroupConfig_terminationPoliciesExplicitDefault = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  max_size = 0
  min_size = 0
  desired_capacity = 0
  termination_policies = ["Default"]

  launch_configuration = "${aws_launch_configuration.foobar.name}"
}
`

const testAccAWSAutoScalingGroupConfig_terminationPoliciesUpdate = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  max_size = 0
  min_size = 0
  desired_capacity = 0
  termination_policies = ["OldestInstance"]

  launch_configuration = "${aws_launch_configuration.foobar.name}"
}
`

const testAccAWSAutoScalingGroupConfig = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_placement_group" "test" {
  name = "test"
  strategy = "cluster"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar3-terraform-test"
  max_size = 5
  min_size = 2
  health_check_grace_period = 300
  health_check_type = "ELB"
  desired_capacity = 4
  force_delete = true
  termination_policies = ["OldestInstance","ClosestToNextInstanceHour"]

  launch_configuration = "${aws_launch_configuration.foobar.name}"

  tag {
    key = "Foo"
    value = "foo-bar"
    propagate_at_launch = true
  }
}
`

const testAccAWSAutoScalingGroupConfigUpdate = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_launch_configuration" "new" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar3-terraform-test"
  max_size = 5
  min_size = 2
  health_check_grace_period = 300
  health_check_type = "ELB"
  desired_capacity = 5
  force_delete = true
  termination_policies = ["ClosestToNextInstanceHour"]

  launch_configuration = "${aws_launch_configuration.new.name}"

  tag {
    key = "Bar"
    value = "bar-foo"
    propagate_at_launch = true
  }
}
`

const testAccAWSAutoScalingGroupConfigWithLoadBalancer = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
	tags { Name = "tf-asg-test" }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group" "foo" {
  vpc_id="${aws_vpc.foo.id}"

  ingress {
    protocol = "-1"
    from_port = 0
    to_port = 0
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    protocol = "-1"
    from_port = 0
    to_port = 0
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_elb" "bar" {
  subnets = ["${aws_subnet.foo.id}"]
	security_groups = ["${aws_security_group.foo.id}"]

  listener {
    instance_port = 80
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  health_check {
    healthy_threshold = 2
    unhealthy_threshold = 2
    target = "HTTP:80/"
    interval = 5
    timeout = 2
  }

	depends_on = ["aws_internet_gateway.gw"]
}

resource "aws_launch_configuration" "foobar" {
  // need an AMI that listens on :80 at boot, this is:
  // bitnami-nginxstack-1.6.1-0-linux-ubuntu-14.04.1-x86_64-hvm-ebs-ami-99f5b1a9-3
  image_id = "ami-b5b3fc85"
  instance_type = "t2.micro"
	security_groups = ["${aws_security_group.foo.id}"]
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["${aws_subnet.foo.availability_zone}"]
	vpc_zone_identifier = ["${aws_subnet.foo.id}"]
  name = "foobar3-terraform-test"
  max_size = 2
  min_size = 2
  health_check_grace_period = 300
  health_check_type = "ELB"
  wait_for_elb_capacity = 2
  force_delete = true

  launch_configuration = "${aws_launch_configuration.foobar.name}"
  load_balancers = ["${aws_elb.bar.name}"]
}
`

const testAccAWSAutoScalingGroupConfigWithAZ = `
resource "aws_vpc" "default" {
  cidr_block = "10.0.0.0/16"
  tags {
     Name = "terraform-test"
  }
}

resource "aws_subnet" "main" {
  vpc_id = "${aws_vpc.default.id}"
  cidr_block = "10.0.1.0/24"
  availability_zone = "us-west-2a"
  tags {
     Name = "terraform-test"
  }
}

resource "aws_launch_configuration" "foobar" {
  image_id = "ami-b5b3fc85"
  instance_type = "t2.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = [
	  "us-west-2a"
  ]
  desired_capacity = 0
  max_size = 0
  min_size = 0
  launch_configuration = "${aws_launch_configuration.foobar.name}"
}
`

const testAccAWSAutoScalingGroupConfigWithVPCIdent = `
resource "aws_vpc" "default" {
  cidr_block = "10.0.0.0/16"
  tags {
     Name = "terraform-test"
  }
}

resource "aws_subnet" "main" {
  vpc_id = "${aws_vpc.default.id}"
  cidr_block = "10.0.1.0/24"
  availability_zone = "us-west-2a"
  tags {
     Name = "terraform-test"
  }
}

resource "aws_launch_configuration" "foobar" {
  image_id = "ami-b5b3fc85"
  instance_type = "t2.micro"
}

resource "aws_autoscaling_group" "bar" {
  vpc_zone_identifier = [
    "${aws_subnet.main.id}",
  ]
  desired_capacity = 0
  max_size = 0
  min_size = 0
  launch_configuration = "${aws_launch_configuration.foobar.name}"
}
`

const testAccAWSAutoScalingGroupConfig_withPlacementGroup = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t2.micro"
}

resource "aws_placement_group" "test" {
  name = "test"
  strategy = "cluster"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar3-terraform-test"
  max_size = 1
  min_size = 1
  health_check_grace_period = 300
  health_check_type = "ELB"
  desired_capacity = 1
  force_delete = true
  termination_policies = ["OldestInstance","ClosestToNextInstanceHour"]
  placement_group = "${aws_placement_group.test.name}"

  launch_configuration = "${aws_launch_configuration.foobar.name}"

  tag {
    key = "Foo"
    value = "foo-bar"
    propagate_at_launch = true
  }
}
`
