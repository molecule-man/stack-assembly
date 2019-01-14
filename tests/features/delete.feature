Feature: stas delete
    Background:
        Given file "cfg.yaml" exists:
            """
            stacks:
                stack1:
                    name: stastest-%scenarioid%
                    path: tpls/stack1.yml
                    tags:
                        STAS_TEST: '%featureid%'

            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
                Cluster:
                    Type: AWS::ECS::Cluster
                    Properties:
                        ClusterName: stastest-%scenarioid%
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        And stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    @wip
    Scenario: I delete non interactively
        When I successfully run "delete -c cfg.yaml --no-interaction"
        Then stack "stastest-%scenarioid%" should not exist
