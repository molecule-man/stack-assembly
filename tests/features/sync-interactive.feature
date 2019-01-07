Feature: stas sync in interactive mode
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

    @short
    Scenario: confirm syncing
        Given I launched "sync -c cfg.yaml --nocolor"
        And terminal shows:
            """
            +--------+-------------------+-------------+--------------------+
            | Action | Resource Type     | Resource ID | Replacement needed |
            +--------+-------------------+-------------+--------------------+
            | Add    | AWS::ECS::Cluster | Cluster     | false              |
            +--------+-------------------+-------------+--------------------+

            *** Commands ***
              [s]ync
              [d]iff
              [q]uit
            What now>
            """
        When I enter "s"
        Then launched program should exit with zero status
        And stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    @short
    Scenario: reject syncing
        Given I launched "sync -c cfg.yaml"
        And terminal shows:
            """
            What now>
            """
        When I enter "q"
        Then terminal shows:
            """
            Interrupted by user
            """
        And terminal shows:
            """
            sync is cancelled
            """
        And launched program should exit with non zero status

    @short
    Scenario: show diff
        Given I launched "sync -c cfg.yaml"
        And terminal shows:
            """
            What now>
            """
        When I enter "d"
        Then terminal shows:
            """
            --- /dev/null
            +++ new-tags/stastest-%scenarioid%
            @@ -0,0 +1 @@
            +STAS_TEST: %featureid%

            --- /dev/null
            +++ new/stastest-%scenarioid%
            @@ -1 +1,5 @@
            -
            +Resources:
            +    Cluster:
            +        Type: AWS::ECS::Cluster
            +        Properties:
            +            ClusterName: stastest-%scenarioid%

            *** Commands ***
              [s]ync
              [d]iff
              [q]uit
            What now>
            """
