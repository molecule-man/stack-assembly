Feature: stas sync with templating

    @short
    Scenario: sync single valid template with templating
        Given file "cfg.yaml" exists:
            """
            parameters:
                Env: "dev"
                Id: "%scenarioid%"
                nameprefix: stastest

            stacks:
                stack1:
                    name: "stack-tpl-{{ .Params.namesuffix }}"
                    path: "tpls/stack1.yml"
                    parameters:
                        namesuffix: "{{ .Params.Env }}-{{ .Params.Id }}"
                    tags:
                        STAS_TEST: "%featureid%"
                        NAME: "{{ .Params.nameprefix }}-{{ .Params.namesuffix }}"
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
                Cluster:
                    Type: AWS::ECS::Cluster
                    Properties:
                        ClusterName: "{{ .Params.nameprefix }}-{{ .Params.namesuffix }}"
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then there should be stack "stack-tpl-dev-%scenarioid%" that matches:
            """
            stackStatus: CREATE_COMPLETE
            resources:
                Cluster: stastest-dev-%scenarioid%
            tags:
                NAME: stastest-dev-%scenarioid%
            """
