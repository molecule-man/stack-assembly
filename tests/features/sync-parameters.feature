Feature: stas sync with parameters

    Scenario: sync single valid template with parameters
        Given file "cfg.toml" exists:
            """
            [parameters]
            Topic1 = "topic1-%scenarioid%"
            Topic2 = "topic2-%scenarioid%"

            [stacks.stack1]
            name = "stack-param-%scenarioid%"
            path = "tpls/stack1.yml"

            [stacks.stack1.tags]
            STAS_TEST = "%featureid%"
            """
        And file "tpls/stack1.yml" exists:
            """
            Parameters:
              Topic1:
                Type: String
              Topic2:
                Type: String
            Resources:
              SNSTopic1:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: !Ref Topic1
              SNSTopic2:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: !Ref Topic2
            """
        When I successfully run "sync -f cfg.toml --no-interaction"
        Then stack "stack-param-%scenarioid%" should have status "CREATE_COMPLETE"
