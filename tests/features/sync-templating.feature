Feature: stas sync with templating
    Scenario: sync single valid template with templating
        Given file "cfg.yaml" exists:
            """
            parameters:
                Env: "dev"
                Id: "%scenarioid%"
                topicprefix: stastest

            stacks:
                stack1:
                    name: "stack-tpl-{{ .Params.namesuffix }}"
                    path: "tpls/stack1.yml"
                    parameters:
                        namesuffix: "{{ .Params.Env }}-{{ .Params.Id }}"
                    tags:
                        STAS_TEST: "%featureid%"
                        TOPIC_NAME: "{{ .Params.topicprefix }}-{{ .Params.namesuffix }}"
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              SNSTopic:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: "{{ .Params.topicprefix }}-{{ .Params.namesuffix }}"
            """
        When I successfully run "sync -f cfg.yaml --no-interaction"
        Then there should be stack "stack-tpl-dev-%scenarioid%" that matches:
            """
            stackStatus: CREATE_COMPLETE
            resources:
                SNSTopic: stastest-dev-%scenarioid%
            tags:
                TOPIC_NAME: stastest-dev-%scenarioid%
            """
