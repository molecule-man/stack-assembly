Feature: stas sync with templating
    Scenario: sync single valid template with templating
        Given file "cfg.yaml" exists:
            """
            parameters:
                Env: "dev"
                Id: "%scenarioid%"

            stacks:
                stack1:
                    name: "stack-tpl-{{ .Params.Env }}-{{ .Params.Id }}"
                    path: "tpls/stack1.yml"
                    tags:
                        STAS_TEST: "%featureid%"
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              SNSTopic:
                Type: AWS::SNS::Topic
                Properties:
                  TopicName: "stastest-{{ .Params.Env }}-{{ .Params.Id }}"
            """
        When I successfully run "sync -f cfg.yaml --no-interaction"
        Then there should be stack "stack-tpl-dev-%scenarioid%" that matches:
            """
            stackStatus: CREATE_COMPLETE
            resources:
                SNSTopic: stastest-dev-%scenarioid%
            """
