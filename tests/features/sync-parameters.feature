Feature: stas sync with parameters

    @short
    Scenario: sync single valid template with parameters
        Given file "cfg.yaml" exists:
            """
            parameters:
              Env: dev
              Topic1: topic1-%scenarioid%
            stacks:
              stack1:
                name: stack-param-%scenarioid%
                path: tpls/stack1.yml
                parameters:
                  Topic2: topic2-%scenarioid%
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Parameters:
              Env:
                Type: String
              Topic1:
                Type: String
              Topic2:
                Type: String
            Resources:
              SNSTopic1:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: !Sub "${Topic1}-${Env}"
              SNSTopic2:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: !Ref Topic2
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stack-param-%scenarioid%" should have status "CREATE_COMPLETE"
