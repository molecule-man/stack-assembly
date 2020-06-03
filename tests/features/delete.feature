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
                        ClusterName: !Ref AWS::StackName
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        And stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    Scenario: I delete non interactively
        When I successfully run "delete -c cfg.yaml --no-interaction"
        Then stack "stastest-%scenarioid%" should not exist

    Scenario: I delete interactively
        Given I launched "delete -c cfg.yaml"
        And terminal shows:
            """
            Stack stastest-%scenarioid% is about to be deleted
            *** Commands ***
              [d]elete
              [a]ll (delete all without asking again)
              [i]nfo (show stack info)
              [s]kip
              [q]uit
            What now>
            """
        When I enter "d"
        Then launched program should exit with zero status
        And stack "stastest-%scenarioid%" should not exist

    @short
    Scenario: I choose to quit while deleting stack
        Given I launched "delete -c cfg.yaml"
        And terminal shows:
            """
            What now>
            """
        When I enter "q"
        Then terminal shows:
            """
            Interrupted by user
            """
        And launched program should exit with non zero status
        And error contains:
            """
            deletion is canceled
            """
        And stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"
