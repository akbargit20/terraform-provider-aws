package aws

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53resolver"
	"github.com/hashicorp/go-multierror"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/aws/internal/service/route53resolver/finder"
	"github.com/hashicorp/terraform-provider-aws/aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func init() {
	resource.AddTestSweepers("aws_route53_resolver_firewall_config", &resource.Sweeper{
		Name: "aws_route53_resolver_firewall_config",
		F:    testSweepRoute53ResolverFirewallConfigs,
		Dependencies: []string{
			"aws_route53_resolver_firewall_config_association",
		},
	})
}

func testSweepRoute53ResolverFirewallConfigs(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).route53resolverconn
	var sweeperErrs *multierror.Error

	err = conn.ListFirewallConfigsPages(&route53resolver.ListFirewallConfigsInput{}, func(page *route53resolver.ListFirewallConfigsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, firewallRuleGroup := range page.FirewallConfigs {
			id := aws.StringValue(firewallRuleGroup.Id)

			log.Printf("[INFO] Deleting Route53 Resolver DNS Firewall config: %s", id)
			r := resourceAwsRoute53ResolverFirewallConfig()
			d := r.Data(nil)
			d.SetId(id)
			err := r.Delete(d, client)

			if err != nil {
				log.Printf("[ERROR] %s", err)
				sweeperErrs = multierror.Append(sweeperErrs, err)
				continue
			}
		}

		return !lastPage
	})
	if testSweepSkipSweepError(err) {
		log.Printf("[WARN] Skipping Route53 Resolver DNS Firewall configs sweep for %s: %s", region, err)
		return sweeperErrs.ErrorOrNil() // In case we have completed some pages, but had errors
	}
	if err != nil {
		sweeperErrs = multierror.Append(sweeperErrs, fmt.Errorf("error retrieving Route53 Resolver DNS Firewall configs: %w", err))
	}

	return sweeperErrs.ErrorOrNil()
}

func TestAccAWSRoute53ResolverFirewallConfig_basic(t *testing.T) {
	var v route53resolver.FirewallConfig
	resourceName := "aws_route53_resolver_firewall_config.test"
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSRoute53Resolver(t) },
		ErrorCheck:   acctest.ErrorCheck(t, route53resolver.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckRoute53ResolverFirewallConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoute53ResolverFirewallConfigConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53ResolverFirewallConfigExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "firewall_fail_open", "ENABLED"),
					acctest.CheckResourceAttrAccountID(resourceName, "owner_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSRoute53ResolverFirewallConfig_disappears(t *testing.T) {
	var v route53resolver.FirewallConfig
	resourceName := "aws_route53_resolver_firewall_config.test"
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSRoute53Resolver(t) },
		ErrorCheck:   acctest.ErrorCheck(t, route53resolver.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckRoute53ResolverFirewallConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoute53ResolverFirewallConfigConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53ResolverFirewallConfigExists(resourceName, &v),
					acctest.CheckResourceDisappears(acctest.Provider, resourceAwsRoute53ResolverFirewallConfig(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckRoute53ResolverFirewallConfigDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*AWSClient).route53resolverconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_resolver_firewall_config" {
			continue
		}

		config, err := finder.FirewallConfigByID(conn, rs.Primary.ID)

		if tfresource.NotFound(err) {
			continue
		}

		if err != nil {
			return err
		}

		if aws.StringValue(config.FirewallFailOpen) == route53resolver.FirewallFailOpenStatusDisabled {
			return nil
		}

		return fmt.Errorf("Route 53 Resolver DNS Firewall config still exists: %s", rs.Primary.ID)
	}

	return nil
}

func testAccCheckRoute53ResolverFirewallConfigExists(n string, v *route53resolver.FirewallConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Route 53 Resolver DNS Firewall config ID is set")
		}

		conn := acctest.Provider.Meta().(*AWSClient).route53resolverconn

		out, err := finder.FirewallConfigByID(conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *out

		return nil
	}
}

func testAccRoute53ResolverFirewallConfigConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_route53_resolver_firewall_config" "test" {
  resource_id        = aws_vpc.test.id
  firewall_fail_open = "ENABLED"
}
`, rName)
}
