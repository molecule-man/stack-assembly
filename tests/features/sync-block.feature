Feature: stas sync block

    Scenario: sync should fail when blocked resource is modified
        Given file "cfg.toml" exists:
            """
            [stacks.stack1]
            name = "stack-block-%scenarioid%"
            path = "tpls/stack1.yml"
            blocked = [ "SNSTopic1" ]

            [stacks.stack1.tags]
            STAS_TEST = "%featureid%"
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              SNSTopic1:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: stastest-%scenarioid%
            """
        When I successfully run "sync -c cfg.toml --no-interaction"
        And I modify file "tpls/stack1.yml":
            """
            Resources:
              SNSTopic1:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: stastest-mod-%scenarioid%
            """
        And I run "sync -c cfg.toml --no-interaction"
        Then exit code should not be zero
        And output should contain:
            """
            does not allow [Update:Replace, Update:Delete]
            """
        And stack "stack-block-%scenarioid%" should have status "UPDATE_ROLLBACK_COMPLETE"
