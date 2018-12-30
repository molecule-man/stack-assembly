Feature: stas sync
    Scenario: sync single valid template without parameters
        Given file "cfg.toml" exists:
            """
            [stacks.stack1]
            name = "stack1-%scenarioid%"
            path = "tpls/stack1.yml"

            [stacks.stack1.tags]
            STAS_TEST = "%featureid%"
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              SNSTopic:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: synctest-%scenarioid%
            """
        When I successfully run "sync -c cfg.toml --no-interaction"
        Then stack "stack1-%scenarioid%" should have status "CREATE_COMPLETE"
