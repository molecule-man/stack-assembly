Feature: stas delete
    Background:
        Given file "cfg.yaml" exists:
            """
            stacks:
                stack1:
                    name: stastest-1-%scenarioid%
                    path: tpls/stack1.yml
                    tags:
                        STAS_TEST: '%featureid%'
                stack2:
                    name: stastest-2-%scenarioid%
                    path: tpls/stack1.yml
                    dependsOn: ["stack1"]
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
        And stack "stastest-1-%scenarioid%" should have status "CREATE_COMPLETE"
        And stack "stastest-2-%scenarioid%" should have status "CREATE_COMPLETE"


    @wip
    Scenario: I choose to delete all
        Given I launched "delete -c cfg.yaml"
        When terminal shows:
            """
            What now>
            """
        And I enter "a"
        Then launched program should exit with zero status
        And stack "stastest-1-%scenarioid%" should not exist
        And stack "stastest-2-%scenarioid%" should not exist

    @wip
    Scenario: I can skip deletion of a stack
        Given I launched "delete -c cfg.yaml"
        When terminal shows:
            """
            Stack stastest-2-%scenarioid% is about to be deleted
            What now>
            """
        And I enter "s"
        Then terminal shows:
            """
            Stack stastest-1-%scenarioid% is about to be deleted
            What now>
            """
        When I enter "d"
        Then launched program should exit with zero status
        And stack "stastest-1-%scenarioid%" should not exist
        And stack "stastest-2-%scenarioid%" should have status "CREATE_COMPLETE"
