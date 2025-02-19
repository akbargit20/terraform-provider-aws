package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccAWSSsmParametersByPathDataSource_basic(t *testing.T) {
	resourceName := "data.aws_ssm_parameters_by_path.test"
	rName1 := acctest.RandomWithPrefix("tf-acc-test")
	rName2 := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, ssm.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsSsmParametersByPathDataSourceConfig(rName1, rName2, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "arns.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "names.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "types.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "with_decryption", "false"),
				),
			},
		},
	})
}

func testAccCheckAwsSsmParametersByPathDataSourceConfig(rName1, rName2 string, withDecryption bool) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "test1" {
  name  = "/%[1]s/param-a"
  type  = "String"
  value = "TestValueA"
}

resource "aws_ssm_parameter" "test2" {
  name  = "/%[1]s/param-b"
  type  = "String"
  value = "TestValueB"
}

resource "aws_ssm_parameter" "test3" {
  name  = "/%[2]s/param-c"
  type  = "String"
  value = "TestValueC"
}

data "aws_ssm_parameters_by_path" "test" {
  path            = "/%[1]s"
  with_decryption = %[3]t

  depends_on = [
    aws_ssm_parameter.test1,
    aws_ssm_parameter.test2,
    aws_ssm_parameter.test3,
  ]
}
`, rName1, rName2, withDecryption)
}
