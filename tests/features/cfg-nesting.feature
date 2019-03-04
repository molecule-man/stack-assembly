Feature: nested

    Scenario: nested stacks (1 level)
        Given file "cfg.yaml" exists:
            """
            stacks:
              nested_stack:
                stacks:
                  stack1:
                    name: stastest-1-%scenarioid%
                    path: tpls/stack1.yml
                    tags:
                      STAS_TEST: '%featureid%'
                  stack2:
                    name: stastest-2-%scenarioid%
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
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-1-%scenarioid%" should have status "CREATE_COMPLETE"
        And stack "stastest-2-%scenarioid%" should have status "CREATE_COMPLETE"

    Scenario: nested stacks (2 levels)
        Given file "cfg.yaml" exists:
            """
            stacks:
              nested_stack:
                stacks:
                  stack1:
                    name: stastest-1-%scenarioid%
                    path: tpls/stack1.yml
                    tags:
                      STAS_TEST: '%featureid%'
                    stacks:
                      stack2:
                        name: stastest-2-%scenarioid%
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
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-1-%scenarioid%" should have status "CREATE_COMPLETE"
        And stack "stastest-2-%scenarioid%" should have status "CREATE_COMPLETE"
