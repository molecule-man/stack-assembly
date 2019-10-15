Feature: stas sync
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

    @short @mock
    Scenario: sync single valid template without parameters
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    @short @mock
    Scenario: sync stack with no changes
        Given I successfully run "sync -c cfg.yaml --no-interaction"
        When I successfully run "sync -c cfg.yaml"
        Then output should contain:
            """
            No changes to be synchronized
            """
