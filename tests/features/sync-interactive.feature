Feature: stas sync in interactive mode
    @short
    Scenario: confirm syncing
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
        And I launched "sync -c cfg.yaml"
        And terminal shows:
            """
            Continue? [Y/n]
            """
        When I enter "y"
        Then launched program should exit with zero status
        And stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    @short
    Scenario: reject syncing
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
        And I launched "sync -c cfg.yaml"
        And terminal shows:
            """
            Continue? [Y/n]
            """
        When I enter "n"
        Then terminal shows:
            """
            Interrupted by user
            """
        And terminal shows:
            """
            sync is cancelled
            """
        And launched program should exit with non zero status
