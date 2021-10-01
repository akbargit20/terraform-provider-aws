package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediastore"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccAWSMediaStoreContainer_basic(t *testing.T) {
	resourceName := "aws_media_store_container.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSMediaStore(t) },
		ErrorCheck:   acctest.ErrorCheck(t, mediastore.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsMediaStoreContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMediaStoreContainerConfig(sdkacctest.RandString(5)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMediaStoreContainerExists(resourceName),
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

func TestAccAWSMediaStoreContainer_tags(t *testing.T) {
	rName := sdkacctest.RandString(5)
	resourceName := "aws_media_store_container.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSMediaStore(t) },
		ErrorCheck:   acctest.ErrorCheck(t, mediastore.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsMediaStoreContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMediaStoreContainerConfigWithTags(rName, "foo", "bar", "fizz", "buzz"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMediaStoreContainerExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "tags.Name", fmt.Sprintf("tf_mediastore_%s", rName)),
					resource.TestCheckResourceAttr(resourceName, "tags.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "tags.fizz", "buzz"),
				),
			},
			{
				Config: testAccMediaStoreContainerConfigWithTags(rName, "foo", "bar2", "fizz2", "buzz2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMediaStoreContainerExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "tags.Name", fmt.Sprintf("tf_mediastore_%s", rName)),
					resource.TestCheckResourceAttr(resourceName, "tags.foo", "bar2"),
					resource.TestCheckResourceAttr(resourceName, "tags.fizz2", "buzz2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccMediaStoreContainerConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMediaStoreContainerExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
		},
	})
}

func testAccCheckAwsMediaStoreContainerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).mediastoreconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_media_store_container" {
			continue
		}

		input := &mediastore.DescribeContainerInput{
			ContainerName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeContainer(input)
		if err != nil {
			if tfawserr.ErrMessageContains(err, mediastore.ErrCodeContainerNotFoundException, "") {
				return nil
			}
			return err
		}

		if *resp.Container.Status != mediastore.ContainerStatusDeleting {
			return fmt.Errorf("Media Store Container (%s) not deleted", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckAwsMediaStoreContainerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*AWSClient).mediastoreconn

		input := &mediastore.DescribeContainerInput{
			ContainerName: aws.String(rs.Primary.ID),
		}

		_, err := conn.DescribeContainer(input)

		return err
	}
}

func testAccPreCheckAWSMediaStore(t *testing.T) {
	conn := testAccProvider.Meta().(*AWSClient).mediastoreconn

	input := &mediastore.ListContainersInput{}

	_, err := conn.ListContainers(input)

	if acctest.PreCheckSkipError(err) {
		t.Skipf("skipping acceptance testing: %s", err)
	}

	if err != nil {
		t.Fatalf("unexpected PreCheck error: %s", err)
	}
}

func testAccMediaStoreContainerConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_media_store_container" "test" {
  name = "tf_mediastore_%s"
}
`, rName)
}

func testAccMediaStoreContainerConfigWithTags(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_media_store_container" "test" {
  name = "tf_mediastore_%[1]s"

  tags = {
    Name = "tf_mediastore_%[1]s"

    %[2]s = %[3]q
    %[4]s = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}
