package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayMethod_basic(t *testing.T) {
	var conf apigateway.Method

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayMethodDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayMethodConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayMethodExists("aws_api_gateway_method.test", &conf),
					testAccCheckAWSAPIGatewayMethodAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_method.test", "http_method", "GET"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_method.test", "authorization", "NONE"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_method.test", "request_models.application/json", "Error"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAPIGatewayMethodConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayMethodExists("aws_api_gateway_method.test", &conf),
					testAccCheckAWSAPIGatewayMethodAttributesUpdate(&conf),
				),
			},
		},
	})
}

func TestAccAWSAPIGatewayMethod_customauthorizer(t *testing.T) {
	var conf apigateway.Method

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayMethodDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayMethodConfigWithCustomAuthorizer,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayMethodExists("aws_api_gateway_method.test", &conf),
					testAccCheckAWSAPIGatewayMethodAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_method.test", "http_method", "GET"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_method.test", "authorization", "CUSTOM"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_method.test", "request_models.application/json", "Error"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAPIGatewayMethodConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayMethodExists("aws_api_gateway_method.test", &conf),
					testAccCheckAWSAPIGatewayMethodAttributesUpdate(&conf),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayMethodAttributes(conf *apigateway.Method) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.HttpMethod != "GET" {
			return fmt.Errorf("Wrong HttpMethod: %q", *conf.HttpMethod)
		}
		if *conf.AuthorizationType != "NONE" && *conf.AuthorizationType != "CUSTOM" {
			return fmt.Errorf("Wrong Authorization: %q", *conf.AuthorizationType)
		}

		if val, ok := conf.RequestParameters["method.request.header.Content-Type"]; !ok {
			return fmt.Errorf("missing Content-Type RequestParameters")
		} else {
			if *val != false {
				return fmt.Errorf("wrong Content-Type RequestParameters value")
			}
		}
		if val, ok := conf.RequestParameters["method.request.querystring.page"]; !ok {
			return fmt.Errorf("missing page RequestParameters")
		} else {
			if *val != true {
				return fmt.Errorf("wrong query string RequestParameters value")
			}
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayMethodAttributesUpdate(conf *apigateway.Method) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.HttpMethod != "GET" {
			return fmt.Errorf("Wrong HttpMethod: %q", *conf.HttpMethod)
		}
		if conf.RequestParameters["method.request.header.Content-Type"] != nil {
			return fmt.Errorf("Content-Type RequestParameters shouldn't exist")
		}
		if val, ok := conf.RequestParameters["method.request.querystring.page"]; !ok {
			return fmt.Errorf("missing updated page RequestParameters")
		} else {
			if *val != false {
				return fmt.Errorf("wrong query string RequestParameters updated value")
			}
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayMethodExists(n string, res *apigateway.Method) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Method ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetMethodInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetMethod(req)
		if err != nil {
			return err
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayMethodDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_method" {
			continue
		}

		req := &apigateway.GetMethodInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		_, err := conn.GetMethod(req)

		if err == nil {
			return fmt.Errorf("API Gateway Method still exists")
		}

		aws2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if aws2err.Code() != "NotFoundException" {
			return err
		}

		return nil
	}

	return nil
}

const testAccAWSAPIGatewayMethodConfigWithCustomAuthorizer = `
resource "aws_api_gateway_rest_api" "test" {
  name = "test"
}

resource "aws_iam_role" "invocation_role" {
  name = "tf_acc_api_gateway_auth_invocation_role"
  path = "/"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "apigateway.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "invocation_policy" {
  name = "default"
  role = "${aws_iam_role.invocation_role.id}"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "lambda:InvokeFunction",
      "Effect": "Allow",
      "Resource": "${aws_lambda_function.authorizer.arn}"
    }
  ]
}
EOF
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "tf_acc_iam_for_lambda_api_gateway_authorizer"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_lambda_function" "authorizer" {
  filename = "test-fixtures/lambdatest.zip"
  source_code_hash = "${base64sha256(file("test-fixtures/lambdatest.zip"))}"
  function_name = "tf_acc_api_gateway_authorizer"
  role = "${aws_iam_role.iam_for_lambda.arn}"
  handler = "exports.example"
}

resource "aws_api_gateway_authorizer" "test" {
  name = "tf-acc-test-authorizer"
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  authorizer_uri = "arn:aws:apigateway:region:lambda:path/2015-03-31/functions/${aws_lambda_function.authorizer.arn}/invocations"
  authorizer_credentials = "${aws_iam_role.invocation_role.arn}"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  parent_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  path_part = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "GET"
  authorization = "CUSTOM"
  authorizer_id = "${aws_api_gateway_authorizer.test.id}"

  request_models = {
    "application/json" = "Error"
  }

  request_parameters_in_json = <<PARAMS
  {
    "method.request.header.Content-Type": false,
	"method.request.querystring.page": true
  }
  PARAMS
}
`

const testAccAWSAPIGatewayMethodConfig = `
resource "aws_api_gateway_rest_api" "test" {
  name = "test"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  parent_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  path_part = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }

  request_parameters_in_json = <<PARAMS
  {
    "method.request.header.Content-Type": false,
	"method.request.querystring.page": true
  }
  PARAMS
}
`

const testAccAWSAPIGatewayMethodConfigUpdate = `
resource "aws_api_gateway_rest_api" "test" {
  name = "test"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  parent_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  path_part = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }

  request_parameters_in_json = <<PARAMS
  {
	"method.request.querystring.page": false
  }
  PARAMS
}
`
