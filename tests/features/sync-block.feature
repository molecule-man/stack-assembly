Feature: stas sync block

    Scenario: sync should fail when blocked resource is modified
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-block-%scenarioid%
                path: tpls/stack1.yml
                blocked:
                  - EcsCluster1
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              EcsCluster1:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest-%scenarioid%
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        And I modify file "tpls/stack1.yml":
            """
            Resources:
              EcsCluster1:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest-mod-%scenarioid%
            """
        And I run "sync -c cfg.yaml --no-interaction"
        Then exit code should not be zero
        And output should contain:
            """
            does not allow [Update:Replace, Update:Delete]
            """
        And stack "stastest-block-%scenarioid%" should have status "UPDATE_ROLLBACK_COMPLETE"
