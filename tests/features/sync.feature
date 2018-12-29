Feature: stas sync
    Scenario: sync single valid template without parameters
        Given file "cfg.toml" exists:
            """
            [stacks.stack1]
            name = "stack1-%testid%"
            path = "tpls/stack1.yml"

            [stacks.stack1.tags]
            STAS_TEST = "true"
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              SNSTopic:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: synctest-%testid%
            """
        When I successfully run "sync -f cfg.toml --no-interaction"
        Then stack "stack1-%testid%" should have status "CREATE_COMPLETE"
