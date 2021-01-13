Feature: stas sync (body test)

    @short
    Scenario: sync stack with embedded body
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-%scenarioid%
                body: |
                  Resources:
                    Cluster:
                      Type: AWS::ECS::Cluster
                      Properties:
                        ClusterName: stastest-%scenarioid%
                tags:
                  STAS_TEST: '%featureid%'
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"


    @short
    Scenario: use templating in body
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-%scenarioid%
                body: |
                  {{ .Params.tpl | printf "%testdir%/%s"  | Exec "cat" }}
                parameters:
                  tpl: tpls/stack1.yml
                tags:
                  STAS_TEST: "%featureid%"
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Ref AWS::StackName
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"
