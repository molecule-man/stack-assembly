Feature: stas sync update

    Scenario: sync create and then update
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-up-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              SNSTopic1:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: stastest-%scenarioid%
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        And I modify file "tpls/stack1.yml":
            """
            Resources:
              SNSTopic1:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: stastest-mod-%scenarioid%
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-up-%scenarioid%" should have status "UPDATE_COMPLETE"
