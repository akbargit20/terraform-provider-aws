package aws

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lexmodelbuildingservice"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	tflex "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/lex"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/lex/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func init() {
	RegisterServiceErrorCheckFunc(lexmodelbuildingservice.EndpointsID, testAccErrorCheckSkipLex)

	resource.AddTestSweepers("aws_lex_bot", &resource.Sweeper{
		Name:         "aws_lex_bot",
		F:            testSweepLexBots,
		Dependencies: []string{"aws_lex_bot_alias"},
	})
}

func testSweepLexBots(region string) error {
	client, err := sharedClientForRegion(region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.(*AWSClient).lexmodelconn
	sweepResources := make([]*testSweepResource, 0)
	var errs *multierror.Error

	input := &lexmodelbuildingservice.GetBotsInput{}

	err = conn.GetBotsPages(input, func(page *lexmodelbuildingservice.GetBotsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, bot := range page.Bots {
			r := resourceAwsLexBot()
			d := r.Data(nil)

			d.SetId(aws.StringValue(bot.Name))

			sweepResources = append(sweepResources, NewTestSweepResource(r, d, client))
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error listing Lex Bot for %s: %w", region, err))
	}

	if err = testSweepResourceOrchestrator(sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Lex Bot for %s: %w", region, err))
	}

	if testSweepSkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Lex Bot sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func testAccErrorCheckSkipLex(t *testing.T) resource.ErrorCheckFunc {
	return testAccErrorCheckSkipMessagesContaining(t,
		"You can't set the enableModelImprovements field to false",
	)
}

func TestAccAwsLexBot_basic(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					testAccCheckAwsLexBotNotExists(testBotID, "1"),

					resource.TestCheckNoResourceAttr(rName, "abort_statement"),
					resource.TestCheckResourceAttrSet(rName, "arn"),
					resource.TestCheckResourceAttrSet(rName, "checksum"),
					resource.TestCheckResourceAttr(rName, "child_directed", "false"),
					resource.TestCheckNoResourceAttr(rName, "clarification_prompt"),
					resource.TestCheckResourceAttr(rName, "create_version", "false"),
					testAccCheckResourceAttrRfc3339(rName, "created_date"),
					resource.TestCheckResourceAttr(rName, "description", "Bot to order flowers on the behalf of a user"),
					resource.TestCheckResourceAttr(rName, "detect_sentiment", "false"),
					resource.TestCheckResourceAttr(rName, "enable_model_improvements", "false"),
					resource.TestCheckResourceAttr(rName, "failure_reason", ""),
					resource.TestCheckResourceAttr(rName, "idle_session_ttl_in_seconds", "300"),
					resource.TestCheckNoResourceAttr(rName, "intent"),
					testAccCheckResourceAttrRfc3339(rName, "last_updated_date"),
					resource.TestCheckResourceAttr(rName, "locale", "en-US"),
					resource.TestCheckResourceAttr(rName, "name", testBotID),
					resource.TestCheckResourceAttr(rName, "nlu_intent_confidence_threshold", "0"),
					resource.TestCheckResourceAttr(rName, "process_behavior", "SAVE"),
					resource.TestCheckResourceAttr(rName, "status", "NOT_BUILT"),
					resource.TestCheckResourceAttr(rName, "version", tflex.LexBotVersionLatest),
					resource.TestCheckNoResourceAttr(rName, "voice_id"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_version_serial(t *testing.T) {
	testCases := map[string]func(t *testing.T){
		"LexBot_createVersion":         testAccAwsLexBot_createVersion,
		"LexBotAlias_botVersion":       testAccAwsLexBotAlias_botVersion,
		"DataSourceLexBot_withVersion": testAccDataSourceAwsLexBot_withVersion,
		"DataSourceLexBotAlias_basic":  testAccDataSourceAwsLexBotAlias_basic,
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			tc(t)
		})
	}
}

func testAccAwsLexBot_createVersion(t *testing.T) {
	var v1, v2 lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	// If this test runs in parallel with other Lex Bot tests, it loses its description
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v1),
					testAccCheckAwsLexBotNotExists(testBotID, "1"),
					resource.TestCheckResourceAttr(rName, "version", tflex.LexBotVersionLatest),
					resource.TestCheckResourceAttr(rName, "description", "Bot to order flowers on the behalf of a user"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_createVersion(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExistsWithVersion(rName, "1", &v2),
					resource.TestCheckResourceAttr(rName, "version", "1"),
					resource.TestCheckResourceAttr(rName, "description", "Bot to order flowers on the behalf of a user"),
				),
			},
		},
	})
}

func TestAccAwsLexBot_abortStatement(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_abortStatement(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "abort_statement.#", "1"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.#", "1"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.0.content", "Sorry, I'm not able to assist at this time"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.0.content_type", "PlainText"),
					resource.TestCheckNoResourceAttr(rName, "abort_statement.0.message.0.group_number"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.response_card", ""),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_abortStatementUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.#", "2"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.0.content", "Sorry, I'm not able to assist at this time"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.0.content_type", "PlainText"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.0.group_number", "1"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.1.content", "Sorry, I'm not able to assist at this time. Good bye."),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.1.content_type", "PlainText"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.message.1.group_number", "1"),
					resource.TestCheckResourceAttr(rName, "abort_statement.0.response_card", "Sorry, I'm not able to assist at this time"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_clarificationPrompt(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_clarificationPrompt(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.#", "1"),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.max_attempts", "2"),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.message.#", "1"),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.message.#", "1"),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.message.0.content", "I didn't understand you, what would you like to do?"),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.message.0.content_type", "PlainText"),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.response_card", ""),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_clarificationPromptUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.max_attempts", "3"),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.message.#", "2"),
					resource.TestCheckResourceAttr(rName, "clarification_prompt.0.response_card", "I didn't understand you, what would you like to do?"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_childDirected(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_childDirectedUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "child_directed", "true"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_description(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_descriptionUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "description", "Bot to order flowers"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_detectSentiment(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_detectSentimentUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "detect_sentiment", "true"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_enableModelImprovements(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_enableModelImprovementsUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "enable_model_improvements", "true"),
					resource.TestCheckResourceAttr(rName, "nlu_intent_confidence_threshold", "0.5"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_idleSessionTtlInSeconds(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_idleSessionTtlInSecondsUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "idle_session_ttl_in_seconds", "600"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_intents(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intentMultiple(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intentMultiple(testBotID),
					testAccAwsLexBotConfig_intentsUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "intent.#", "2"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_computeVersion(t *testing.T) {
	var v1 lexmodelbuildingservice.GetBotOutput
	var v2 lexmodelbuildingservice.GetBotAliasOutput

	botResourceName := "aws_lex_bot.test"
	botAliasResourceName := "aws_lex_bot_alias.test"
	intentResourceName := "aws_lex_intent.test"
	intentResourceName2 := "aws_lex_intent.test_2"

	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	version := "1"
	updatedVersion := "2"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_createVersion(testBotID),
					testAccAwsLexBotConfig_intentMultiple(testBotID),
					testAccAwsLexBotAliasConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExistsWithVersion(botResourceName, version, &v1),
					resource.TestCheckResourceAttr(botResourceName, "version", version),
					resource.TestCheckResourceAttr(botResourceName, "intent.#", "1"),
					resource.TestCheckResourceAttr(botResourceName, "intent.0.intent_version", version),
					testAccCheckAwsLexBotAliasExists(botAliasResourceName, &v2),
					resource.TestCheckResourceAttr(botAliasResourceName, "bot_version", version),
					resource.TestCheckResourceAttr(intentResourceName, "version", version),
				),
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intentMultipleSecondUpdated(testBotID),
					testAccAwsLexBotConfig_multipleIntentsWithVersion(testBotID),
					testAccAwsLexBotAliasConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExistsWithVersion(botResourceName, updatedVersion, &v1),
					resource.TestCheckResourceAttr(botResourceName, "version", updatedVersion),
					resource.TestCheckResourceAttr(botResourceName, "intent.#", "2"),
					resource.TestCheckResourceAttr(botResourceName, "intent.0.intent_version", version),
					resource.TestCheckResourceAttr(botResourceName, "intent.1.intent_version", updatedVersion),
					resource.TestCheckResourceAttr(botAliasResourceName, "bot_version", updatedVersion),
					resource.TestCheckResourceAttr(intentResourceName, "version", version),
					resource.TestCheckResourceAttr(intentResourceName2, "version", updatedVersion),
				),
			},
		},
	})
}

func TestAccAwsLexBot_locale(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_localeUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "locale", "en-GB"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_voiceId(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_voiceIdUpdate(testBotID),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					resource.TestCheckResourceAttr(rName, "voice_id", "Justin"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsLexBot_disappears(t *testing.T) {
	var v lexmodelbuildingservice.GetBotOutput
	rName := "aws_lex_bot.test"
	testBotID := "test_bot_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(lexmodelbuildingservice.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, lexmodelbuildingservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsLexBotDestroy,
		Steps: []resource.TestStep{
			{
				Config: composeConfig(
					testAccAwsLexBotConfig_intent(testBotID),
					testAccAwsLexBotConfig_basic(testBotID),
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLexBotExists(rName, &v),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsLexBot(), rName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAwsLexBotExistsWithVersion(rName, botVersion string, v *lexmodelbuildingservice.GetBotOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("Not found: %s", rName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Lex Bot ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lexmodelconn

		output, err := finder.BotVersionByName(conn, rs.Primary.ID, botVersion)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccCheckAwsLexBotExists(rName string, output *lexmodelbuildingservice.GetBotOutput) resource.TestCheckFunc {
	return testAccCheckAwsLexBotExistsWithVersion(rName, tflex.LexBotVersionLatest, output)
}

func testAccCheckAwsLexBotNotExists(botName, botVersion string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).lexmodelconn

		_, err := finder.BotVersionByName(conn, botName, botVersion)

		if tfresource.NotFound(err) {
			return nil
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("Lex Box %s/%s still exists", botName, botVersion)
	}
}

func testAccCheckAwsLexBotDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).lexmodelconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_lex_bot" {
			continue
		}

		output, err := conn.GetBotVersions(&lexmodelbuildingservice.GetBotVersionsInput{
			Name: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if output == nil || len(output.Bots) == 0 {
			return nil
		}

		return fmt.Errorf("Lex Bot %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccAwsLexBotConfig_intent(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_intent" "test" {
  create_version = true
  name           = "%s"
  fulfillment_activity {
    type = "ReturnIntent"
  }
  sample_utterances = [
    "I would like to pick up flowers",
  ]
}
`, rName)
}

func testAccAwsLexBotConfig_intentMultiple(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_intent" "test" {
  create_version = true
  name           = "%[1]s"
  fulfillment_activity {
    type = "ReturnIntent"
  }
  sample_utterances = [
    "I would like to pick up flowers",
  ]
}

resource "aws_lex_intent" "test_2" {
  create_version = true
  name           = "%[1]stwo"
  fulfillment_activity {
    type = "ReturnIntent"
  }
  sample_utterances = [
    "I would like to pick up flowers",
  ]
}
`, rName)
}

func testAccAwsLexBotConfig_intentMultipleSecondUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_intent" "test" {
  create_version = true
  name           = "%[1]s"
  fulfillment_activity {
    type = "ReturnIntent"
  }
  sample_utterances = [
    "I would like to pick up flowers",
  ]
}

resource "aws_lex_intent" "test_2" {
  create_version = true
  name           = "%[1]stwo"
  fulfillment_activity {
    type = "ReturnIntent"
  }
  sample_utterances = [
    "I would like to return these flowers",
  ]
}
`, rName)
}

func testAccAwsLexBotConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = false
  description    = "Bot to order flowers on the behalf of a user"
  name           = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_createVersion(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed   = false
  create_version   = true
  description      = "Bot to order flowers on the behalf of a user"
  name             = "%s"
  process_behavior = "BUILD"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_abortStatement(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = false
  description    = "Bot to order flowers on the behalf of a user"
  name           = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_abortStatementUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = false
  description    = "Bot to order flowers on the behalf of a user"
  name           = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
      group_number = 1
    }
    message {
      content      = "Sorry, I'm not able to assist at this time. Good bye."
      content_type = "PlainText"
      group_number = 1
    }
    response_card = "Sorry, I'm not able to assist at this time"
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_clarificationPrompt(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = false
  description    = "Bot to order flowers on the behalf of a user"
  name           = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  clarification_prompt {
    max_attempts = 2
    message {
      content      = "I didn't understand you, what would you like to do?"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_clarificationPromptUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = false
  description    = "Bot to order flowers on the behalf of a user"
  name           = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  clarification_prompt {
    max_attempts = 3
    message {
      content      = "I didn't understand you, what would you like to do?"
      content_type = "PlainText"
      group_number = 1
    }
    message {
      content      = "I didn't understand you, can you re-phrase your request, please?"
      content_type = "PlainText"
      group_number = 1
    }
    response_card = "I didn't understand you, what would you like to do?"
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_childDirectedUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = true
  description    = "Bot to order flowers on the behalf of a user"
  name           = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_descriptionUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = false
  description    = "Bot to order flowers"
  name           = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_detectSentimentUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed   = false
  description      = "Bot to order flowers on the behalf of a user"
  detect_sentiment = true
  name             = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_enableModelImprovementsUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed                  = false
  description                     = "Bot to order flowers on the behalf of a user"
  enable_model_improvements       = true
  name                            = "%s"
  nlu_intent_confidence_threshold = 0.5
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_idleSessionTtlInSecondsUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed              = false
  description                 = "Bot to order flowers on the behalf of a user"
  idle_session_ttl_in_seconds = 600
  name                        = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_intentsUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = false
  description    = "Bot to order flowers on the behalf of a user"
  name           = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
  intent {
    intent_name    = aws_lex_intent.test_2.name
    intent_version = aws_lex_intent.test_2.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_localeUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed            = false
  description               = "Bot to order flowers on the behalf of a user"
  enable_model_improvements = true
  locale                    = "en-GB"
  name                      = "%s"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_voiceIdUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed = false
  description    = "Bot to order flowers on the behalf of a user"
  name           = "%s"
  voice_id       = "Justin"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
}
`, rName)
}

func testAccAwsLexBotConfig_multipleIntentsWithVersion(rName string) string {
	return fmt.Sprintf(`
resource "aws_lex_bot" "test" {
  child_directed   = false
  create_version   = true
  description      = "Bot to order flowers on the behalf of a user"
  name             = "%s"
  process_behavior = "BUILD"
  abort_statement {
    message {
      content      = "Sorry, I'm not able to assist at this time"
      content_type = "PlainText"
    }
  }
  intent {
    intent_name    = aws_lex_intent.test.name
    intent_version = aws_lex_intent.test.version
  }
  intent {
    intent_name    = aws_lex_intent.test_2.name
    intent_version = aws_lex_intent.test_2.version
  }
}
`, rName)
}
